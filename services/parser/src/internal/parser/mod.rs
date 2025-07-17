//! Main HTML parsing module.
//!
//! Provides the `HtmlParser` that extracts structured data from HTML pages
//! including title, meta tags, main content, links, images, headings, etc.

use anyhow::Result;
use scraper::Html;

mod extractors;
mod language_detector;
pub mod models;
mod text_utils;

use extractors::{
    extract_canonical_url, extract_headings, extract_images, extract_links, extract_main_content,
    extract_meta_description, extract_meta_keywords, extract_title,
};
use language_detector::{detect_language, map_lang_to_pg};
use models::ParsedPage;

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

        let title = extract_title(&document);
        let description = extract_meta_description(&document);
        let meta_keywords = extract_meta_keywords(&document);
        let canonical_url = extract_canonical_url(&document);

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
}
