# typed: false
# frozen_string_literal: true

class AddForeignKeyToEditSuggestionPagesLatestRevision < ActiveRecord::Migration[8.0]
  def change
    add_foreign_key :edit_suggestion_pages, :edit_suggestion_page_revisions, column: :latest_revision_id, validate: false
  end
end
