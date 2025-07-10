//! HTML extractors for structured data.
//!
//! This module provides functions to extract specific pieces of information from
//! an HTML document, including headings, links, images, and main content.

use once_cell::sync::Lazy;
use scraper::{Html, Selector};
use readability::extractor;
use std::io::Cursor;
use url::Url;

use crate::models::{ImageData, LinkData};
use super::text_utils::clean_text;

// Precompiled selectors for performance

/// Selector for headings <h1>-<h6>
static HEADING_SELECTOR: Lazy<Selector> =
    Lazy::new(|| Selector::parse("h1, h2, h3, h4, h5, h6").unwrap());

/// Selector for <a href>
static LINK_SELECTOR: Lazy<Selector> =
    Lazy::new(|| Selector::parse("a[href]").unwrap());

/// Selector for <img src>
static IMG_SELECTOR: Lazy<Selector> =
    Lazy::new(|| Selector::parse("img[src]").unwrap());


/// Extracts and cleans all `<h1>`–`<h6>` headings from the document.
pub fn extract_headings(document: &Html) -> Vec<String> {
    document
        .select(&HEADING_SELECTOR)
        .map(|element| clean_text(&element.text().collect::<String>()))
        .filter(|text| !text.is_empty())
        .collect()
}

/// Extracts all `<a>` links, resolving relative URLs and marking external links.
///
/// Skips links with `javascript:` or `mailto:` schemes or empty text.
///
/// # Arguments
/// - `document`: Parsed HTML document.
/// - `base_url`: URL of the page, used to resolve relative links.
///
/// # Returns
/// A vector of `LinkData`.
pub fn extract_links(document: &Html, base_url: &str) -> Vec<LinkData> {
    let base = Url::parse(base_url).ok();

    document
        .select(&LINK_SELECTOR)
        .filter_map(|element| {
            let href = element.value().attr("href")?;
            let text = clean_text(&element.text().collect::<String>());

            if href.starts_with("javascript:") || href.starts_with("mailto:") || text.is_empty() {
                return None;
            }

            let resolved_url = if let Some(base) = &base {
                base.join(href).map(|u| u.to_string()).unwrap_or_else(|_| href.to_string())
            } else {
                href.to_string()
            };

            let is_external = if let (Ok(base_url), Ok(link_url)) =
                (Url::parse(base_url), Url::parse(&resolved_url))
            {
                base_url.domain() != link_url.domain()
            } else {
                false
            };

            Some(LinkData {
                url: resolved_url,
                text,
                is_external,
            })
        })
        .collect()
}

/// Extracts all `<img>` elements, resolving relative `src` attributes.
///
/// # Arguments
/// - `document`: Parsed HTML document.
/// - `base_url`: URL of the page, used to resolve relative image URLs.
///
/// # Returns
/// A vector of `ImageData`.
pub fn extract_images(document: &Html, base_url: &str) -> Vec<ImageData> {
    let base = Url::parse(base_url).ok();

    document
        .select(&IMG_SELECTOR)
        .filter_map(|element| {
            let src = element.value().attr("src")?;
            let alt = element.value().attr("alt").map(|s| s.to_string());
            let title = element.value().attr("title").map(|s| s.to_string());

            let resolved_src = if let Some(base) = &base {
                base.join(src).map(|u| u.to_string()).unwrap_or_else(|_| src.to_string())
            } else {
                src.to_string()
            };

            Some(ImageData {
                src: resolved_src,
                alt,
                title,
            })
        })
        .collect()
}

/// Extracts the main readable content from the page using `readability`.
///
/// # Arguments
/// - `document`: Parsed HTML document.
/// - `base_url`: The URL of the page, used by readability.
///
/// # Returns
/// Cleaned main content text, or empty string if extraction fails.
pub fn extract_main_content(document: &Html, base_url: &str) -> String {
    // Get the original HTML as a string
    let html_str = document.root_element().html();

    // Create a BufRead from the HTML string
    let mut reader = Cursor::new(html_str);

    // Parse the base URL
    let url = match Url::parse(base_url) {
        Ok(u) => u,
        Err(_) => return String::new(),
    };

    // Run readability
    match extractor::extract(&mut reader, &url) {
        Ok(article) => {
            // Convert readable HTML content to plain text
            let doc = Html::parse_fragment(&article.content);
            clean_text(&doc.root_element().text().collect::<String>())
        }
        Err(_) => String::new(),
    }
}