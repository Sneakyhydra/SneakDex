//! Main HTML parsing module.
//!
//! Provides the `HtmlParser` that extracts structured data from HTML pages
//! including title, meta tags, main content, links, images, headings, etc.

use anyhow::Result;
use scraper::{Html, Selector};

mod extractors;
mod language_detector;
mod text_utils;

use extractors::{extract_headings, extract_images, extract_links, extract_main_content};
use language_detector::detect_language;
use text_utils::clean_text;

use crate::config::Config;
use crate::models::ParsedPage;

/// HTML parser that extracts structured data from a page.
///
/// Initialized with a `Config` to enforce content limits & settings.
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
        let schema_data = self.extract_schema_data(&document);

        // main content now requires the URL for readability
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

        // fallback full body text (not readability-cleaned)
        let body_text = self.extract_body_text(&document);

        Ok(ParsedPage {
            url: url.to_string(),
            title,
            description,
            body_text,
            cleaned_text,
            headings,
            links,
            images,
            canonical_url,
            language,
            word_count,
            meta_keywords,
            schema_data,
            timestamp: chrono::Utc::now(),
            content_type: "text/html".to_string(),
            encoding: "utf-8".to_string(),
            reading_time: None,
            og_tags: None,
            twitter_cards: None,
            robots_meta: None,
            readability_score: None,
            additional_metadata: std::collections::HashMap::new(),
        })
    }

    /// Extracts the `<title>` tag.
    fn extract_title(&self, document: &Html) -> String {
        static TITLE_SELECTOR: once_cell::sync::Lazy<Selector> =
            once_cell::sync::Lazy::new(|| Selector::parse("title").unwrap());

        document
            .select(&TITLE_SELECTOR)
            .next()
            .map(|e| clean_text(&e.inner_html()))
            .unwrap_or_else(|| "No Title".to_string())
    }

    /// Extracts `<meta name="description">`.
    fn extract_meta_description(&self, document: &Html) -> Option<String> {
        static DESC_SELECTOR: once_cell::sync::Lazy<Selector> =
            once_cell::sync::Lazy::new(|| Selector::parse("meta[name='description']").unwrap());

        document
            .select(&DESC_SELECTOR)
            .next()
            .and_then(|e| e.value().attr("content"))
            .map(clean_text)
    }

    /// Extracts `<meta name="keywords">`.
    fn extract_meta_keywords(&self, document: &Html) -> Option<String> {
        static KEYWORDS_SELECTOR: once_cell::sync::Lazy<Selector> =
            once_cell::sync::Lazy::new(|| Selector::parse("meta[name='keywords']").unwrap());

        document
            .select(&KEYWORDS_SELECTOR)
            .next()
            .and_then(|e| e.value().attr("content"))
            .map(clean_text)
    }

    /// Extracts `<link rel="canonical">`.
    fn extract_canonical_url(&self, document: &Html) -> Option<String> {
        static CANONICAL_SELECTOR: once_cell::sync::Lazy<Selector> =
            once_cell::sync::Lazy::new(|| Selector::parse("link[rel='canonical']").unwrap());

        document
            .select(&CANONICAL_SELECTOR)
            .next()
            .and_then(|e| e.value().attr("href"))
            .map(|href| href.to_string())
    }

    /// Extracts JSON-LD `<script type="application/ld+json">`.
    fn extract_schema_data(&self, document: &Html) -> Option<String> {
        static SCHEMA_SELECTOR: once_cell::sync::Lazy<Selector> =
            once_cell::sync::Lazy::new(|| {
                Selector::parse("script[type='application/ld+json']").unwrap()
            });

        document
            .select(&SCHEMA_SELECTOR)
            .next()
            .map(|e| e.inner_html())
    }

    /// Fallback: extracts all visible text from `<body>`.
    fn extract_body_text(&self, document: &Html) -> String {
        static BODY_SELECTOR: once_cell::sync::Lazy<Selector> =
            once_cell::sync::Lazy::new(|| Selector::parse("body").unwrap());

        document
            .select(&BODY_SELECTOR)
            .next()
            .map(|e| clean_text(&e.text().collect::<String>()))
            .unwrap_or_default()
    }
}
