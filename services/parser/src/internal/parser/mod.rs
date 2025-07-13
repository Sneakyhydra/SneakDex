//! Main HTML parsing module.
//!
//! Provides the `HtmlParser` that extracts structured data from HTML pages
//! including title, meta tags, main content, links, images, headings, etc.

use anyhow::Result;
use once_cell::sync::Lazy;
use scraper::{Html, Selector};

mod extractors;
mod language_detector;
pub mod models;
mod text_utils;

use extractors::{extract_headings, extract_images, extract_links, extract_main_content};
use language_detector::{detect_language, map_lang_to_pg};
use models::ParsedPage;
use text_utils::clean_text;

use crate::internal::config::Config;

/// HTML parser that extracts structured data from a page.
///
/// Initialized with a `Config` to enforce content limits & settings.
#[derive(Clone)]
pub struct HtmlParser {
    config: Config,
}

impl HtmlParser {
    /// Creates a new `HtmlParser`.
    pub fn new(config: &Config) -> Self {
        Self {
            config: config.clone(),
        }
    }

    /// Parses HTML and returns a `ParsedPage` result.
    ///
    /// Validates content size, extracts all fields, and ensures minimum content length.
    pub fn parse_html(&self, html: &str, url: &str) -> Result<ParsedPage> {
        // Enforce max content length
        if html.len() > self.config.max_content_length {
            return Err(anyhow::anyhow!("Content too large: {} bytes", html.len()));
        }

        let document = Html::parse_document(html);

        let title = self.extract_title(&document);
        let description = self.extract_meta_description(&document);
        let meta_keywords = self.extract_meta_keywords(&document);
        let canonical_url = self.extract_canonical_url(&document);

        let cleaned_text = extract_main_content(&document, url);

        // Validate minimum content length
        if cleaned_text.len() < self.config.min_content_length {
            return Err(anyhow::anyhow!(
                "Content too short: {} characters",
                cleaned_text.len()
            ));
        }

        let headings = extract_headings(&document);
        let links = extract_links(&document, url);
        let images = extract_images(&document, url);

        let word_count = cleaned_text.split_whitespace().count();
        let language = detect_language(&cleaned_text);
        let pg_lang = language.as_deref().map(map_lang_to_pg).unwrap_or("simple");

        Ok(ParsedPage {
            url: url.to_string(),
            title,
            description,
            cleaned_text,
            headings,
            links,
            images,
            canonical_url,
            language: Some(pg_lang.to_string()),
            word_count,
            meta_keywords,
            timestamp: chrono::Utc::now(),
            content_type: "text/html".to_string(),
            encoding: "utf-8".to_string(),
        })
    }

    /// Extracts the `<title>` tag.
    fn extract_title(&self, document: &Html) -> String {
        static TITLE_SELECTOR: Lazy<Selector> = Lazy::new(|| Selector::parse("title").unwrap());

        document
            .select(&TITLE_SELECTOR)
            .next()
            .map(|e| clean_text(&e.inner_html()))
            .unwrap_or_else(|| "No Title".to_string())
    }

    /// Extracts `<meta name="description">`.
    fn extract_meta_description(&self, document: &Html) -> Option<String> {
        static DESC_SELECTOR: Lazy<Selector> =
            Lazy::new(|| Selector::parse("meta[name='description']").unwrap());

        document
            .select(&DESC_SELECTOR)
            .next()
            .and_then(|e| e.value().attr("content"))
            .map(clean_text)
    }

    /// Extracts `<meta name="keywords">`.
    fn extract_meta_keywords(&self, document: &Html) -> Option<String> {
        static KEYWORDS_SELECTOR: Lazy<Selector> =
            Lazy::new(|| Selector::parse("meta[name='keywords']").unwrap());

        document
            .select(&KEYWORDS_SELECTOR)
            .next()
            .and_then(|e| e.value().attr("content"))
            .map(clean_text)
    }

    /// Extracts `<link rel="canonical">`.
    fn extract_canonical_url(&self, document: &Html) -> Option<String> {
        static CANONICAL_SELECTOR: Lazy<Selector> =
            Lazy::new(|| Selector::parse("link[rel='canonical']").unwrap());

        document
            .select(&CANONICAL_SELECTOR)
            .next()
            .and_then(|e| e.value().attr("href"))
            .map(|href| href.to_string())
    }
}
