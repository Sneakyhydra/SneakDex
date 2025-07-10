//! Data models for parsed HTML pages.
//!
//! These models represent the structured data extracted from an HTML page, including
//! text, links, images, metadata, and more. All models implement `Serialize` to support
//! easy serialization (e.g., to JSON).

use serde::Serialize;
use std::collections::HashMap;

/// Represents an image (`<img>`) found on the page.
#[derive(Debug, Serialize)]
pub struct ImageData {
    /// The `src` attribute (URL) of the image.
    pub src: String,

    /// The `alt` attribute of the image, if present.
    pub alt: Option<String>,

    /// The `title` attribute of the image, if present.
    pub title: Option<String>,
}

/// Represents a hyperlink (`<a>`) found on the page.
#[derive(Debug, Serialize)]
pub struct LinkData {
    /// The `href` URL of the link.
    pub url: String,

    /// The visible text of the link.
    pub text: String,

    /// Whether the link is external to the page's domain.
    pub is_external: bool,
}

/// Represents Open Graph meta tags.
#[derive(Debug, Serialize)]
pub struct OpenGraphData {
    /// Open Graph title
    pub title: Option<String>,
    /// Open Graph description
    pub description: Option<String>,
    /// Open Graph image URL
    pub image: Option<String>,
    /// Open Graph type
    pub r#type: Option<String>,
    /// Open Graph URL
    pub url: Option<String>,
    /// Additional Open Graph properties
    pub additional: HashMap<String, String>,
}

/// Represents Twitter Card meta tags.
#[derive(Debug, Serialize)]
pub struct TwitterCardData {
    /// Twitter Card type
    pub card: Option<String>,
    /// Twitter Card title
    pub title: Option<String>,
    /// Twitter Card description
    pub description: Option<String>,
    /// Twitter Card image
    pub image: Option<String>,
    /// Twitter Card creator
    pub creator: Option<String>,
    /// Additional Twitter Card properties
    pub additional: HashMap<String, String>,
}

/// Represents a fully-parsed HTML page and its extracted data.
#[derive(Debug, Serialize)]
pub struct ParsedPage {
    /// The URL of the page.
    pub url: String,

    /// The page's `<title>`.
    pub title: String,

    /// The page's meta description, if present.
    pub description: Option<String>,

    /// The full text content of the page.
    pub body_text: String,

    /// Cleaned and normalized text content.
    pub cleaned_text: String,

    /// A list of headings (`<h1>`, `<h2>`, etc.) found on the page.
    pub headings: Vec<String>,

    /// All hyperlinks (`<a>`) found on the page.
    pub links: Vec<LinkData>,

    /// All images (`<img>`) found on the page.
    pub images: Vec<ImageData>,

    /// The canonical URL of the page, if specified.
    pub canonical_url: Option<String>,

    /// Detected language of the page, if determined.
    pub language: Option<String>,

    /// Word count of the `cleaned_text`.
    pub word_count: usize,

    /// The page's meta keywords, if present.
    pub meta_keywords: Option<String>,

    /// Any structured data (`schema.org`) found, as JSON-LD or similar.
    pub schema_data: Option<String>,

    /// Timestamp when this page was parsed.
    pub timestamp: chrono::DateTime<chrono::Utc>,

    /// Content type of the page
    pub content_type: String,

    /// Character encoding of the page
    pub encoding: String,

    /// Estimated reading time in minutes
    pub reading_time: Option<u32>,

    /// Open Graph meta tags
    pub og_tags: Option<OpenGraphData>,

    /// Twitter Card meta tags
    pub twitter_cards: Option<TwitterCardData>,

    /// Robots meta tag content
    pub robots_meta: Option<String>,

    /// Flesch reading ease score (0-100)
    pub readability_score: Option<f32>,

    /// Additional metadata as key-value pairs
    pub additional_metadata: HashMap<String, String>,
}
