# typed: false
# frozen_string_literal: true

class ValidateAddForeignKeyToEditSuggestionPagesLatestRevision < ActiveRecord::Migration[8.0]
  def change
    validate_foreign_key :edit_suggestion_pages, :edit_suggestion_page_revisions
  end
end
