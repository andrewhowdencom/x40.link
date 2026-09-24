// Listing an owner's links across domains queries the "id" collection group.
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
