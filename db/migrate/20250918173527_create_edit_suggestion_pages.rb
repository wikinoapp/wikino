# typed: false
# frozen_string_literal: true

class CreateEditSuggestionPages < ActiveRecord::Migration[8.0]
  def change
    create_table :edit_suggestion_pages, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.uuid :space_id, null: false
      t.uuid :edit_suggestion_id, null: false
      t.uuid :page_id
      t.string :title_before
      t.string :title_after
      t.text :body_before
      t.text :body_after
      t.timestamps

      t.index :space_id
      t.index :edit_suggestion_id
      t.index :page_id
      t.index [:edit_suggestion_id, :page_id], unique: true
    end

    add_foreign_key :edit_suggestion_pages, :spaces, column: :space_id, validate: false
    add_foreign_key :edit_suggestion_pages, :edit_suggestions, column: :edit_suggestion_id, validate: false
    add_foreign_key :edit_suggestion_pages, :pages, column: :page_id, validate: false
  end
end
