// Retain the legacy index so the offline migration can be rolled back.
resource "google_firestore_field" "link_owner" {
  project    = data.google_project.project.project_id
  collection = "id"
  field      = "owner"

  index_config {
    indexes {
      order       = "ASCENDING"
      query_scope = "COLLECTION"
    }

    indexes {
      order       = "DESCENDING"
      query_scope = "COLLECTION"
    }

    indexes {
      order       = "ASCENDING"
      query_scope = "COLLECTION_GROUP"
    }
  }
}

// Encoded source paths use a separate collection group.
resource "google_firestore_field" "link_owner_encoded" {
  project    = data.google_project.project.project_id
  collection = "shortLinks"
  field      = "owner"

  index_config {
    indexes {
      order       = "ASCENDING"
      query_scope = "COLLECTION"
    }
    indexes {
      order       = "DESCENDING"
      query_scope = "COLLECTION"
    }
    indexes {
      order       = "ASCENDING"
      query_scope = "COLLECTION_GROUP"
    }
  }
}
