package types

import (
	"encoding/xml"
)

// Core Discogs entities - Corrected
type Artist struct {
	XMLName     xml.Name `xml:"artist"`
	ID          int64    `xml:"id"`
	Name        string   `xml:"name"`
	RealName    string   `xml:"realname"`
	Profile     string   `xml:"profile"`
	DataQuality string   `xml:"data_quality"`
	URLs        string   `xml:"urls>url"`
	// NameVariationsstring   `xml:"namevariations>name"`
	Aliases Alias  `xml:"aliases>alias"`
	Members Member `xml:"members>member"`
	Groups  Group  `xml:"groups>group"`
	Images  Image  `xml:"images>image"`
}

type Label struct {
	XMLName     xml.Name `xml:"label"`
	ID          int64    `xml:"id"`
	Name        string   `xml:"name"`
	ContactInfo string   `xml:"contactinfo"`
	Profile     string   `xml:"profile"`
	DataQuality string   `xml:"data_quality"`
	ParentLabel string   `xml:"parentLabel"`
	SubLabels   string   `xml:"sublabels>label"`
	URLs        string   `xml:"urls>url"`
	Images      Image    `xml:"images>image"`
}

type Release struct {
	XMLName     xml.Name        `xml:"release"`
	ID          int64           `xml:"id,attr"`
	Status      string          `xml:"status"`
	Title       string          `xml:"title"`
	Country     string          `xml:"country"`
	Released    string          `xml:"released"` // Keep as string for inconsistent formats
	Notes       string          `xml:"notes"`
	DataQuality string          `xml:"data_quality"`
	MasterID    int64           `xml:"master_id"`
	Artists     []ReleaseArtist `xml:"artists>artist"`
	Labels      []ReleaseLabel  `xml:"labels>label"`
	Formats     Format          `xml:"formats>format"`
	Genres      []string        `xml:"genres>genre"`
	Styles      []string        `xml:"styles>style"`
	Tracklist   []Track         `xml:"tracklist>track"`
	Images      []Image         `xml:"images>image"`
}

type Master struct {
	XMLName     xml.Name       `xml:"master"`
	ID          int64          `xml:"id,attr"`
	MainRelease int            `xml:"main_release"`
	Title       string         `xml:"title"`
	Year        int            `xml:"year"`
	Notes       string         `xml:"notes"`
	DataQuality string         `xml:"data_quality"`
	Artists     []MasterArtist `xml:"artists>artist"`
	Genres      []string       `xml:"genres>genre"`
	Styles      []string       `xml:"styles>style"`
	Videos      Video          `xml:"videos>video"`
	Images      Image          `xml:"images>image"`
}

// Supporting structs - Corrected
type Alias struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:",chardata"`
}

type Member struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:",chardata"`
}

type Group struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:",chardata"`
}

type Format struct {
	Name string `xml:"name,attr"`
	Qty  string `xml:"qty,attr"`
	Text string `xml:"text,attr"`
	// Descriptionsstring `xml:"descriptions>description"`
}

type Track struct {
	Position string `xml:"position"`
	Title    string `xml:"title"`
	Duration string `xml:"duration"`
}

type Video struct {
	Duration int    `xml:"duration,attr"`
	Embed    bool   `xml:"embed,attr"`
	Source   string `xml:"src,attr"`
	Title    string `xml:"title"`
}

// New specialized structs for nested elements with additional attributes
type ReleaseArtist struct {
	ID   int64  `xml:"id"`
	Name string `xml:"name"`
	Anv  string `xml:"anv"`
	Join string `xml:"join"`
	Role string `xml:"role"`
}

type ReleaseLabel struct {
	Name        string `xml:"name,attr"`
	Catno       string `xml:"catno,attr"`
	ID          int64  `xml:"id,attr"`
	Sublabel    bool   `xml:"sublabel,attr"`
	ParentLabel int64  `xml:"parent_label_id,attr"`
}

type MasterArtist struct {
	ID   int64  `xml:"id"`
	Name string `xml:"name"`
	Anv  string `xml:"anv"`
	Join string `xml:"join"`
	Role string `xml:"role"`
}

// Additional common structs from the schema
type Image struct {
	URI    string `xml:"uri,attr"`
	URI150 string `xml:"uri150,attr"`
	Type   string `xml:"type,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
}

type Identifier struct {
	Type        string `xml:"type,attr"`
	Value       string `xml:"value,attr"`
	Description string `xml:"description,attr"`
}

type Company struct {
	ID             int    `xml:"id,attr"`
	Name           string `xml:"name"`
	Catno          string `xml:"catno"`
	EntityType     int    `xml:"entity_type"`
	EntityTypeName string `xml:"entity_type_name"`
}

// This is for reference only and the full report
// Technical Report: Validation and Refactoring of a Discogs Data Dump Ingestion Tool1.0 Executive SummaryThe analysis of the provided Go programming language structs, intended for parsing the Discogs monthly data dumps, indicates a solid foundational understanding of the project's requirements. However, a detailed technical validation reveals that the structs are not "100% accurate" for successfully unmarshaling the entirety of the XML-formatted data. The current design contains critical structural and data type mismatches that would lead to unmarshaling failures and data loss when applied to the actual data files.The primary conclusion is that a fundamental conceptual discrepancy exists between the XML format of the data dumps and the JSON format used by the Discogs API. This has resulted in an incorrect mapping of hierarchical XML elements and attributes to flat Go structs. Specifically, elements such as aliases, members, and groups are improperly structured, preventing the capture of essential ID and Name data. Furthermore, the inherent scale of the data dumps, with files exceeding several gigabytes in size, poses a significant architectural challenge. A standard unmarshaling approach would be highly susceptible to memory exhaustion, rendering the tool non-viable for production use.This report provides a comprehensive, field-by-field breakdown of the discrepancies found within the user's code. Following this detailed analysis, a complete, refactored set of production-grade Go structs is provided. The report concludes with a blueprint for a robust data pipeline architecture that utilizes a high-performance streaming XML decoder. This approach addresses the issues of structural mapping, data inconsistencies, and memory management, providing a resilient and scalable solution for processing the Discogs data dumps. The recommendations are grounded in established data engineering principles and supported by empirical evidence from existing projects that successfully handle this exact dataset.2.0 Foundational Review: Deciphering the Discogs Data Ecosystem2.1 The Critical Distinction: Data Dumps vs. API EndpointsA precise understanding of the data's source and format is paramount to building a successful ingestion tool. The monthly Discogs data dumps are distributed as large, gzipped XML files from a dedicated S3 bucket.1 This is a critical point of divergence from the Discogs API, which is described as a RESTful interface that provides JSON-formatted information for various database objects, including artists, releases, and labels.2 While the data dumps and the API contain information on the same entities, the underlying serialization format and structural schema are distinct.The Discogs data dump page asserts that the XML format is "formatted according to the API spec".3 This statement can be a source of significant confusion. It does not imply that the data dump is a direct XML conversion of the JSON API response. Instead, it suggests that the two sources share a common conceptual data model. The XML schema, being an older and more verbose format, is inherently more hierarchical and less flat than the modern JSON payloads returned by the API. A developer creating Go structs based on a mental model of the API's JSON output would inevitably produce an incorrect schema for the XML source. The user's provided code, while correctly using Go's XML tags, exhibits this exact conceptual misalignment, which is the root cause of the identified structural flaws. This analysis is supported by the fact that various third-party projects, such as those on Kaggle, must first process the raw XML before converting it into more accessible formats like CSV or JSON for general use.4 This demonstrates that the native XML format requires a specialized parsing strategy.2.2 The Challenge of Scale: Why a Naive Approach FailsThe Discogs data dumps are of a substantial size, with the gzipped artists file alone weighing in at 418.0 MB.1 Once decompressed, these files become a significant computational challenge, easily exceeding several gigabytes. The user's included DiscogsArtists, DiscogsLabels, DiscogsReleases, and DiscogsMasters structs, which serve as root containers for a slice of the primary entity, imply a standard unmarshaling approach. This would involve reading the entire XML file into memory before processing it.However, such a strategy is not viable for a production-grade tool. Loading a multi-gigabyte XML file into memory for parsing would consume massive amounts of RAM and, in most environments, lead to a program failure due to memory exhaustion (an "out of memory" or OOM error). A review of existing solutions for this problem confirms this assessment. A Python-based parser, for instance, is noted for its ability to process a 6.0 GB gzipped XML file in just over an hour while maintaining a remarkably low memory footprint of only 17 MB.7 Similarly, an open-source Go package designed for this specific task explicitly employs a block-based decoding strategy to process data in manageable chunks, avoiding the need to load the entire dataset at once.8 These examples provide definitive evidence that the correct architectural pattern for this project is a streaming XML decoder, which processes the file element by element rather than attempting to hold the entire structure in memory. This approach is a fundamental requirement for a solution that is both "100% accurate" and performant.3.0 Detailed Technical Validation: A Field-by-Field Analysis of the Proposed Go Structs3.1 MethodologyThe following is a structured, field-by-field review of the user's provided Go structs. The analysis identifies correct mappings, incomplete definitions, and fundamental structural flaws that would impede a successful unmarshaling process. The findings are based on a technical interpretation of how Go's encoding/xml package operates in conjunction with the likely schema of the Discogs XML dumps, as inferred from external data sources and examples.3.2 Analysis of the Artist Struct (imports.Artist)ID, Name, RealName, Profile: These fields are correctly defined as simple int and string types, with the xml tags pointing to their respective element names. For a basic artist entry, these mappings are sound.URLs and NameVars: The xml:"urls>url" and xml:"namevariations>name" tags are also correct. This syntax effectively tells the xml decoder to look inside the <urls> and <namevariations> parent elements and collect all child elements named <url> and <name> into a slice of strings (string). This correctly captures a simple list of values.Aliases, Members, Groups: This is where a critical structural misinterpretation occurs. The user has defined nested structs (Alias, Member, Group) for these fields, but the xml tag for the slices is xml:"aliases>name". A review of external datasets derived from the raw dumps, such as the Kaggle datasets which contain columns for both aliases_name and aliases_name_id, indicates a more complex underlying XML structure.4 It is highly probable that the XML for an alias looks like <alias id="1234">John Doe</alias> or, more likely for this schema, <alias id="1234"><name>John Doe</name></alias>. The user's struct Alias is defined as ID int xml:"id", and Name string xml:",chardata". The tag xml:"aliases>name" on the Artist struct's Aliases field attempts to unmarshal a slice of Alias structs from a child element named <name>. The id field within the Alias struct would then attempt to look for an <id> child element within <name>, which does not exist in the likely schema. This fundamental mismatch would prevent the unmarshaling of any alias data. The correct approach requires a different XML tag on the Artist struct (xml:"aliases>alias") and an adjusted Alias struct that accurately maps the id as an attribute or a direct child element. The same logic and structural errors apply to the Members and Groups fields.3.3 Analysis of the Label Struct (imports.Label)ID, Name, ContactInfo, Profile, URLs: These fields are correctly defined and mapped for unmarshaling the respective elements.ParentLabel and SubLabels: These are plausible mappings. The ParentLabel field likely corresponds to a simple element with a string value. The SubLabels field, with the xml:"sublabels>label" tag, correctly expects a list of simple string elements, which is a common pattern for lists in this schema.3.4 Analysis of the Release Struct (imports.Release)Released field: The user has correctly defined Released as a string to handle the incoming XML data. However, external analysis of the data dump reveals that the released field is not always in a standardized format.7 While some dates follow the YYYY-MM-DD pattern, others may be just YYYY or contain invalid characters or placeholders. A naive attempt to unmarshal this into a time.Time struct would cause the program to crash on the first malformed date. Keeping the field as a string is the correct approach for robust unmarshaling, but it necessitates a subsequent, separate data validation and conversion step within the data pipeline to handle these inconsistencies. This is a crucial point of concern for achieving a truly "100% accurate" ingestion process.Artists and Labels: The user has reused the main Artist and Label structs for these nested elements, which again represents a structural flaw. Within the context of a release, the <artist> and <label> elements often contain additional, context-specific data that is not present in the top-level entity dumps. For example, a release artist might have a join element specifying their role (A.K.A., featuring), or a release label might have a catalogno attribute. By reusing the main structs, the unmarshaler would ignore these critical contextual fields, leading to an incomplete and inaccurate representation of the data. For a complete solution, new, specialized structs for ReleaseArtist and ReleaseLabel are required to capture these additional details.3.5 Analysis of the Master Struct (imports.Master)Videos: The user's Video struct correctly uses xml:"duration,attr", xml:"embed,attr", and xml:"src,attr" tags to capture attributes of the <video> element. However, the struct also includes a URI field without a corresponding xml tag. This field is redundant, as the video's URI is already being captured by the Source field, which correctly maps to the src attribute. This is a minor but notable inconsistency in the struct definition.4.0 Comprehensive Recommendations for a Robust Data Pipeline4.1 Proposed Refactored Go StructsTo address the discrepancies identified in the previous section, a complete set of corrected and production-ready structs is provided below. These structs are designed to accurately map the complex XML hierarchy and capture all available data, including attributes and nested elements.Gopackage imports
//
// import (
// 	"encoding/xml"
// 	"time"
// )
//
// // Core Discogs entities - Corrected
// type Artist struct {
// 	XMLName      xml.Name     `xml:"artist"`
// 	ID           int          `xml:"id"`
// 	Name         string       `xml:"name"`
// 	RealName     string       `xml:"realname"`
// 	Profile      string       `xml:"profile"`
// 	DataQuality  string       `xml:"data_quality"`
// 	URLs        string     `xml:"urls>url"`
// 	NameVariationsstring   `xml:"namevariations>name"`
// 	Aliases     Alias      `xml:"aliases>alias"`
// 	Members     Member     `xml:"members>member"`
// 	Groups      Group      `xml:"groups>group"`
// 	Images      Image      `xml:"images>image"`
// }
//
// type Label struct {
// 	XMLName     xml.Name `xml:"label"`
// 	ID          int      `xml:"id"`
// 	Name        string   `xml:"name"`
// 	ContactInfo string   `xml:"contactinfo"`
// 	Profile     string   `xml:"profile"`
// 	DataQuality string   `xml:"data_quality"`
// 	ParentLabel string   `xml:"parentLabel"`
// 	SubLabels  string `xml:"sublabels>label"`
// 	URLs       string `xml:"urls>url"`
// 	Images     Image  `xml:"images>image"`
// }
//
// type Release struct {
// 	XMLName     xml.Name      `xml:"release"`
// 	ID          int           `xml:"id"`
// 	Status      string        `xml:"status"`
// 	Title       string        `xml:"title"`
// 	Country     string        `xml:"country"`
// 	Released    string        `xml:"released"` // Keep as string for inconsistent formats
// 	Notes       string        `xml:"notes"`
// 	DataQuality string        `xml:"data_quality"`
// 	MasterID    int           `xml:"master_id"`
// 	Artists    ReleaseArtist `xml:"artists>artist"`
// 	Labels     ReleaseLabel  `xml:"labels>label"`
// 	Formats    Format      `xml:"formats>format"`
// 	Genres     string      `xml:"genres>genre"`
// 	Styles     string      `xml:"styles>style"`
// 	Tracklist  Track       `xml:"tracklist>track"`
// 	Images     Image       `xml:"images>image"`
// 	IdentifiersIdentifier  `xml:"identifiers>identifier"`
// 	Companies  Company     `xml:"companies>company"`
// }
//
// type Master struct {
// 	XMLName     xml.Name       `xml:"master"`
// 	ID          int            `xml:"id"`
// 	MainRelease int            `xml:"main_release"`
// 	Title       string         `xml:"title"`
// 	Year        int            `xml:"year"`
// 	Notes       string         `xml:"notes"`
// 	DataQuality string         `xml:"data_quality"`
// 	Artists    MasterArtist `xml:"artists>artist"`
// 	Genres     string       `xml:"genres>genre"`
// 	Styles     string       `xml:"styles>style"`
// 	Videos     Video        `xml:"videos>video"`
// 	Images     Image        `xml:"images>image"`
// }
//
// // Supporting structs - Corrected
// type Alias struct {
// 	ID   int    `xml:"id,attr"`
// 	Name string `xml:",chardata"`
// }
//
// type Member struct {
// 	ID   int    `xml:"id,attr"`
// 	Name string `xml:",chardata"`
// }
//
// type Group struct {
// 	ID   int    `xml:"id,attr"`
// 	Name string `xml:",chardata"`
// }
//
// type Format struct {
// 	Name         string   `xml:"name,attr"`
// 	Qty          string   `xml:"qty,attr"`
// 	Text         string   `xml:"text,attr"`
// 	Descriptionsstring `xml:"descriptions>description"`
// }
//
// type Track struct {
// 	Position string `xml:"position"`
// 	Title    string `xml:"title"`
// 	Duration string `xml:"duration"`
// }
//
// type Video struct {
// 	Duration int    `xml:"duration,attr"`
// 	Embed    bool   `xml:"embed,attr"`
// 	Source   string `xml:"src,attr"`
// 	Title    string `xml:"title"`
// }
//
// // New specialized structs for nested elements with additional attributes
// type ReleaseArtist struct {
// 	ID   int    `xml:"id,attr"`
// 	Name string `xml:"name"`
// 	Anv  string `xml:"anv"`
// 	Join string `xml:"join"`
// 	Role string `xml:"role"`
// }
//
// type ReleaseLabel struct {
// 	Name      string `xml:"name,attr"`
// 	Catno     string `xml:"catno,attr"`
// 	ID        int    `xml:"id,attr"`
// 	Sublabel  bool   `xml:"sublabel,attr"`
// 	ParentLabel int  `xml:"parent_label_id,attr"`
// }
//
// type MasterArtist struct {
// 	ID    int    `xml:"id,attr"`
// 	Name  string `xml:"name"`
// 	Anv   string `xml:"anv"`
// 	Join  string `xml:"join"`
// 	Role  string `xml:"role"`
// }
//
// // Additional common structs from the schema
// type Image struct {
// 	URI      string `xml:"uri,attr"`
// 	URI150   string `xml:"uri150,attr"`
// 	Type     string `xml:"type,attr"`
// 	Width    int    `xml:"width,attr"`
// 	Height   int    `xml:"height,attr"`
// }
//
// type Identifier struct {
// 	Type        string `xml:"type,attr"`
// 	Value       string `xml:"value,attr"`
// 	Description string `xml:"description,attr"`
// }
//
// type Company struct {
// 	ID          int    `xml:"id,attr"`
// 	Name        string `xml:"name"`
// 	Catno       string `xml:"catno"`
// 	EntityType  int    `xml:"entity_type"`
// 	EntityTypeName string `xml:"entity_type_name"`
// }
// 4.2 A Streaming XML Parser ArchitectureThe only practical approach for processing multi-gigabyte files is a streaming parser. Go's encoding/xml package provides the xml.NewDecoder for this purpose. The following steps outline the recommended architecture:Open the Compressed File: Begin by opening the gzipped data dump file.Create a Decompression Reader: Instantiate a gzip.Reader using the file as its input. This handles the decompression transparently as data is read from the stream.Initialize the XML Decoder: Create a new xml.Decoder instance, pointing it to the gzip.Reader.Iterate and Decode: Use a for loop to read tokens from the decoder one by one. The decoder.Token() method returns the next XML token in the stream.Target the Primary Elements: Within the loop, check if the current token is a StartElement with the name of the top-level entity being processed (e.g., artist, label, release, master).Unmarshaling a Single Element: Once a target StartElement is found, use decoder.DecodeElement() to unmarshal that single element and its children directly into the appropriate, corrected Go struct. This process consumes only the portion of the file needed for that single record, avoiding the memory overhead of a full file unmarshal.Process and Persist: Immediately after a single element is unmarshaled, the resulting struct can be processed, validated, and persisted to a database or other storage medium. This block-based processing is a key architectural choice for handling large datasets efficiently.8 The json and db tags in the user's original structs suggest a clear intention to move this data to another system. By processing each record individually, the pipeline can operate with a minimal memory footprint.This streaming architecture is not merely a performance enhancement; it is a prerequisite for a functional tool given the scale of the data. It directly addresses the memory constraints and allows for the development of a resilient, fault-tolerant pipeline.4.3 Resilient Data Handling: Validation & Error ManagementA production-grade tool must be able to gracefully handle data quality issues. A Python parser for this same dataset notes that some dump files contain malformed XML due to forbidden control characters.7 This underscores the need for robust error management.The streaming parser approach allows for isolated error handling. If a single element fails to unmarshal, the error can be logged, and the loop can continue to the next valid element, preventing a single malformed record from crashing the entire ingestion job. This is in contrast to a full-file unmarshal, which would fail on the first error and terminate the process. The data-discogs Go library provides an excellent example of this, logging successful and failed processing blocks, which is essential for tracking progress and diagnosing issues in a massive data run.8Specific data validation is also necessary. For the Released field, which is kept as a string to avoid unmarshaling errors, a separate validation function should be created. This function can use regular expressions or a series of conditional checks to parse the date into a more structured type, like time.Time, while gracefully handling malformed or incomplete values.5.0 Appendix and Reference TablesTable 1: Go Struct Validation & Correction TableOriginal User's Code SnippetCorrected Go Code SnippetDetailed Rationaletype Artist struct {... AliasesAlias xml:"aliases>name"... }
// type Alias struct { ID int xml:"id", Name string xml:",chardata" }type Artist struct {... AliasesAlias xml:"aliases>alias"... }
// type Alias struct { ID int xml:"id,attr", Name string xml:",chardata" }The XML for an alias element likely contains an id attribute, and the name is a child node. The tag xml:"aliases>name" is incorrect and would not capture the id. The corrected tag xml:"aliases>alias" is required on the Artist struct. The Alias struct is corrected to map the ID to an XML attribute with xml:"id,attr". This change is essential for capturing the ID data, which is confirmed to exist by third-party data extractions.4type Release struct {... ArtistsArtist xml:"artists>artist"... LabelsLabel xml:"labels>label"... }`type Release struct {... ArtistsReleaseArtist xml:"artists>artist"... LabelsReleaseLabel xml:"labels>label"... }
// // New structs for context-specific data
// type ReleaseArtist struct { ID int xml:"id,attr", Name string xml:"name", Join string xml:"join" }Reusing the core Artist and Label structs for nested elements is a design flaw. Within a release, an artist or label element often contains additional contextual information, such as a join string (Join) or a catalog number (Catno), which the top-level structs lack. Creating dedicated structs (ReleaseArtist, ReleaseLabel) with the correct tags ensures that all available contextual data is captured for a complete and accurate record of the release.type Release struct {... Released string xml:"released"... }type Release struct {... Released string xml:"released"... }The data type is correct. The recommendation is to keep this field as a string to prevent unmarshaling errors, as the released date in the source data is known to have inconsistent formats (e.g., YYYY vs. YYYY-MM-DD).7 A separate, defensive parsing step should be implemented after unmarshaling to validate and convert the string into a structured date format.type Video struct { Duration int xml:"duration,attr", Embed bool xml:"embed,attr", Source string xml:"src,attr", Title string xml:"title", Description string xml:"description", URI string xml:"uri"}`type Video struct { Duration int xml:"duration,attr", Embed bool xml:"embed,attr", Source string xml:"src,attr", Title string xml:"title" }The URI field is redundant. The video's source URL is already correctly mapped by the Source field from the src attribute. Removing the superfluous field simplifies the struct and clarifies its purpose.Table 2: Comprehensive Discogs XML Schema ReferenceTop-Level EntityPrimary Elements & Attributesartist<id>, <name>, <realname>, <profile>, <data_quality>, <images>, <urls>, <namevariations>, <aliases>, <members>, <groups>label<id>, <name>, <contactinfo>, <profile>, <data_quality>, <images>, <urls>, <parentLabel>, <sublabels>release<id>, <status>, <title>, <country>, <released>, <notes>, <data_quality>, <master_id>, <artists>, <labels>, <formats>, <genres>, <styles>, <tracklist>, <images>, <identifiers>, <companies>master<id>, <main_release>, <title>, <year>, <notes>, <data_quality>, <artists>, <genres>, <styles>, <videos>, <images>Nested ElementsKey Elements & Attributesalias, member, group<id attribute>, text contentrelease artist<id attribute>, <name>, <anv>, <join>, <role>release label<id attribute>, <name attribute>, <catno attribute>track<position>, <title>, <duration>format<name attribute>, <qty attribute>, <text attribute>, <descriptions>video<duration attribute>, <embed attribute>, <src attribute>, <title>image<uri attribute>, <uri150 attribute>, <type attribute>, <width attribute>, <height attribute>identifier<type attribute>, <value attribute>, <description attribute>company<id attribute>, <name>, <catno>, <entity_type>, <entity_type_name>
