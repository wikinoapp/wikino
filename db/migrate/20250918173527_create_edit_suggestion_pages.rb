# typed: false
# frozen_string_literal: true

class CreateEditSuggestionPages < ActiveRecord::Migration[8.0]
  def change
    create_table :edit_suggestion_pages, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, type: :uuid, null: false, foreign_key: true
      t.references :edit_suggestion, type: :uuid, null: false, foreign_key: true
      t.references :page, type: :uuid, foreign_key: true
      t.references :page_revision, type: :uuid, foreign_key: true
      t.citext :title, null: false
      t.citext :body, null: false
      t.timestamps

      t.index %i[edit_suggestion_id page_id], unique: true
    end
  end
end
