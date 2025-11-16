# Core Architecture: The "Tags'n Links" System

## 1. Core Philosophy: Pragmatism Over Dogma

The system is a hybrid: a robust relational database (MariaDB) provides persistence and integrity, while the Go application code provides the intelligence and business logic. We build on battle-tested, standard technologies to create a powerful, flexible, and maintainable solution. The "smarts" are in our code.

The user interface will be delivered via a **Server-Side Rendering (SSR)** model. The Go backend will be responsible for rendering HTML pages. To provide a modern, dynamic user experience without the complexity of a full frontend framework, we will use **Fixi.js** to add AJAX-powered interactivity to these server-rendered pages.

The application itself is divided into two distinct interfaces: the **Object Management UI**, which allows users to create, edit, and link instances of `pieces`, `contlets`, and `tags`; and the **Class Management UI**, which allows administrators to define and modify the very structure of those `Classes` by adding or removing fields.

## 2. Core Concepts: `Objects` and `Classes`

The system's data model is built on two fundamental concepts: `Classes` and `Objects`.

*   A **`Class`** is the *definition* or *template* for a type of data. For example, `contlet_paragraph` is a `Class` that defines the structure for all paragraphs, specifying that they must have a `text_content` field. The structure of these `Classes` can be modified by an administrator through the **Class Management UI**.

*   An **`Object`** is a specific *instance* of a `Class`. A single paragraph with the text "Hello, World" is an `Object` belonging to the `contlet_paragraph` `Class`. Users create and manage `Objects` through the **Object Management UI**.

This distinction is the foundation of the entire system.

## 3. The Data Model: A `Class`-based Architecture

The architecture for our data model defines how we build `Classes` using database tables, and how we create `Objects` by adding rows to those tables. The system uses a central `entity` table to ensure every `Object` has a unique identity, regardless of its `Class`.

An `Object`'s identity and data are stored across two types of tables:

*   **The `entity` Table:** A single, central table whose sole purpose is to generate a globally unique ID for every `Object` in the system.
*   **`Class` Tables:** Each `Class` (e.g., `contlet_paragraph`, `content_piece`, `tag`) is physically represented by its own dedicated database table. This table contains all of the data fields specific to that `Class`. The primary key of a `Class` table is also a foreign key to the `entity` table, linking the `Object`'s specific data to its unique global identity.

---

## 4. Final Schema: MariaDB Implementation

-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- LAYER 0: THE ENTITY CORE
-- Provides a globally unique ID for every Object in the system.
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

CREATE TABLE entity (
    id INT PRIMARY KEY AUTO_INCREMENT
) ENGINE=InnoDB;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- LAYER 1: THE STRUCTURAL CORE (The Content "Nodes")
-- Defines the raw, addressable pieces of content.
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- A controlled vocabulary for the structural kinds of Contlet Classes.
CREATE TABLE contlet_class (
    name VARCHAR(255) PRIMARY KEY,
    description TEXT NOT NULL
) ENGINE=InnoDB;

-- A composed assembly of Contlet Objects, representing a final output like a blog post.
CREATE TABLE content_piece (
    id INT PRIMARY KEY, -- FK to entity.id
    class VARCHAR(255) NOT NULL, -- e.g., 'blog_post', 'tweet'
    title TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    FOREIGN KEY (id) REFERENCES entity(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- The ordered assembly of Contlet Objects into a Content Piece Object.
CREATE TABLE content_piece_contlets (
    content_piece_id INT NOT NULL REFERENCES content_piece(id) ON DELETE CASCADE,
    contlet_id INT NOT NULL REFERENCES entity(id) ON DELETE RESTRICT, -- Prevent deleting a contlet Object that is in use.
    sort_order INT NOT NULL, -- Use spaced integers (100, 200, 300) for easy reordering.
    PRIMARY KEY (content_piece_id, sort_order)
) ENGINE=InnoDB;

-- SPECIFIC CONTLET CLASS TABLES --

CREATE TABLE contlet_paragraph (
    id INT PRIMARY KEY, -- FK to entity.id
    text_content TEXT NOT NULL,
    FOREIGN KEY (id) REFERENCES entity(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE contlet_image (
    id INT PRIMARY KEY, -- FK to entity.id
    src VARCHAR(1024) NOT NULL,
    alt_text TEXT,
    width INT,
    height INT,
    FOREIGN KEY (id) REFERENCES entity(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE contlet_heading (
    id INT PRIMARY KEY, -- FK to entity.id
    text_content VARCHAR(1024) NOT NULL,
    level INT NOT NULL DEFAULT 2 CHECK (level BETWEEN 1 AND 6),
    FOREIGN KEY (id) REFERENCES entity(id) ON DELETE CASCADE
) ENGINE=InnoDB;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- LAYER 2: THE SEMANTIC LAYER (The "Tags")
-- The descriptive metadata that makes content discoverable.
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- The high-level categories for tags (e.g., "General Keywords", "Programming Languages").
CREATE TABLE taxonomy (
    id INT PRIMARY KEY, -- FK to entity.id
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    FOREIGN KEY (id) REFERENCES entity(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- The specific Tag Objects. Each Tag Object MUST belong to a Taxonomy Object.
CREATE TABLE tag (
    id INT PRIMARY KEY, -- FK to entity.id
    taxonomy_id INT NOT NULL REFERENCES taxonomy(id) ON DELETE RESTRICT,
    value VARCHAR(255) NOT NULL,
    -- Ensures a tag's value is unique within its taxonomy.
    UNIQUE (taxonomy_id, value),
    FOREIGN KEY (id) REFERENCES entity(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Links any entity `Object` (Contlets, Content Pieces, etc.) to Tag Objects.
CREATE TABLE entity_tags (
    entity_id INT NOT NULL REFERENCES entity(id) ON DELETE CASCADE,
    tag_id INT NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
    PRIMARY KEY (entity_id, tag_id)
) ENGINE=InnoDB;


-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
-- LAYER 3: THE RELATIONAL LAYER (The "Links")
-- The generalized knowledge graph connecting all `Objects`.
-- ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

-- A controlled vocabulary for the "verbs" or types of relationships.
CREATE TABLE link_class (
    name VARCHAR(255) PRIMARY KEY,
    description TEXT NOT NULL,
    -- Optional: define symmetric relationships, e.g., 'related_to'
    symmetric_link VARCHAR(255) REFERENCES link_class(name)
) ENGINE=InnoDB;

-- The heart of the knowledge graph. Links any `Object` to any other `Object`.
CREATE TABLE entity_relationships (
    id INT PRIMARY KEY AUTO_INCREMENT,
    -- The "Subject" of the link (the "from" node)
    subject_id INT NOT NULL REFERENCES entity(id) ON DELETE CASCADE,
    -- The "Verb" of the link
    link_type VARCHAR(255) NOT NULL REFERENCES link_class(name),
    -- The "Object" of the link (the "to" node)
    object_id INT NOT NULL REFERENCES entity(id) ON DELETE CASCADE,
    -- Optional metadata about the link itself
    source TEXT,
    confidence REAL CHECK (confidence BETWEEN 0.0 AND 1.0),
    -- Ensures a specific link between two `Objects` is not duplicated.
    UNIQUE (subject_id, link_type, object_id)
) ENGINE=InnoDB;

## 5. Application & UI Design

### 5.1. Routing Philosophy

The application's routes are defined by a specific, explicit list of URIs that correspond directly to user actions. This RPC-style (Remote Procedure Call) approach is chosen for clarity and directness, rather than adhering to a strict RESTful model. The full list of routes is the canonical guide for the application's API surface.

### 5.2. Dynamic Class Management

A core feature of this system is the ability for an administrator to modify the schema of a `Class` (e.g., `content_piece`, `contlet_paragraph`) directly from the **Class Management UI**. This is achieved through the careful, controlled use of `ALTER TABLE` commands.

To ensure both user-friendliness and security, the process is as follows:

1.  **Semantic Field Types:** The UI will not expose raw SQL data types. Instead, a user will select a semantic type from a simple dropdown menu (e.g., "Single Line of Text", "Paragraph", "Number", "Date").
2.  **Backend Mapping:** The Go backend maintains a non-negotiable, internal map that translates these semantic types into safe, specific SQL data types (e.g., "Single Line of Text" maps to `VARCHAR(255)`).
3.  **Secure Execution:** The backend constructs the `ALTER TABLE` query using the pre-defined SQL type from its internal map. All user-provided input (like the new field name) is strictly validated to prevent SQL injection. This allows for the required UI-driven flexibility while eliminating the risks associated with exposing raw DDL commands.
