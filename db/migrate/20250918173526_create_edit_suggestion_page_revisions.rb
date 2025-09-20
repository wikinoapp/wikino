# typed: false
# frozen_string_literal: true

class CreateEditSuggestionPageRevisions < ActiveRecord::Migration[8.0]
  def change
    create_table :edit_suggestion_page_revisions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, type: :uuid, null: false, foreign_key: true
      t.references :edit_suggestion_page, type: :uuid, null: false, foreign_key: true
      t.references :editor_space_member, type: :uuid, null: false, foreign_key: {to_table: :space_members}
      t.citext :title, null: false
      t.citext :body, null: false
      t.text :body_html, null: false
      t.timestamps

      t.index %i[edit_suggestion_page_id created_at]
    end
  end
end