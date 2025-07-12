//! HTML extractors for structured data.
//!
//! This module provides functions to extract specific pieces of information from
//! an HTML document, including headings, links, images, and main content.

use once_cell::sync::Lazy;
use readability::extractor;
use scraper::{Html, Selector};
use std::io::Cursor;
use url::Url;

use super::models::{Heading, ImageData, LinkData};
use super::text_utils::clean_text;

// Precompiled selectors for performance

/// Selector for headings h1 - h6
static HEADING_SELECTOR: Lazy<Selector> =
    Lazy::new(|| Selector::parse("h1, h2, h3, h4, h5, h6").unwrap());

/// Selector for "a href"
static LINK_SELECTOR: Lazy<Selector> = Lazy::new(|| Selector::parse("a[href]").unwrap());

/// Selector for "img src"
static IMG_SELECTOR: Lazy<Selector> = Lazy::new(|| Selector::parse("img[src]").unwrap());

/// Selector for "body" fallback
static BODY_SELECTOR: Lazy<Selector> = Lazy::new(|| Selector::parse("body").unwrap());

/// Extracts and cleans all `<h1>`–`<h6>` headings from the document.
pub fn extract_headings(document: &Html) -> Vec<Heading> {
    document
        .select(&HEADING_SELECTOR)
        .filter_map(|element| {
            let tag_name = element.value().name(); // e.g. "h1"
            let level = tag_name.strip_prefix('h')?.parse::<u8>().ok()?;
            let text = clean_text(&element.text().collect::<String>());
            if text.is_empty() {
                return None;
            }
            Some(Heading { level, text })
        })
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
                base.join(href)
                    .map(|mut u| {
                        u.set_fragment(None);
                        u
                    })
                    .unwrap_or_else(|_| Url::parse(href).unwrap_or_else(|_| base.clone()))
            } else {
                Url::parse(href).unwrap_or_else(|_| Url::parse("about:blank").unwrap())
            };

            let resolved_url_str = resolved_url.to_string();

            let is_external =
                if let (Some(base), Ok(link_url)) = (base.clone(), Url::parse(&resolved_url_str)) {
                    base.domain() != link_url.domain()
                } else {
                    false
                };

            Some(LinkData {
                url: resolved_url_str,
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
                base.join(src)
                    .map(|mut u| {
                        u.set_fragment(None);
                        u
                    })
                    .unwrap_or_else(|_| Url::parse(src).unwrap_or_else(|_| base.clone()))
            } else {
                Url::parse(src).unwrap_or_else(|_| Url::parse("about:blank").unwrap())
            };

            Some(ImageData {
                src: resolved_src.to_string(),
                alt,
                title,
            })
        })
        .collect()
}

/// Extracts the main readable content from the page using `readability`.
///
/// If readability fails, falls back to body text.
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
    if let Ok(article) = extractor::extract(&mut reader, &url) {
        let doc = Html::parse_fragment(&article.content);
        let text = clean_text(&doc.root_element().text().collect::<String>());
        if !text.is_empty() {
            return text;
        }
    }

    // Fallback to raw body text
    if let Some(body) = document.select(&BODY_SELECTOR).next() {
        return clean_text(&body.text().collect::<String>());
    }

    String::new()
}
