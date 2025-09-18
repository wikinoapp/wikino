# typed: false
# frozen_string_literal: true

class CreateEditSuggestionComments < ActiveRecord::Migration[8.0]
  def change
    create_table :edit_suggestion_comments, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.uuid :space_id, null: false
      t.uuid :edit_suggestion_id, null: false
      t.uuid :created_user_id, null: false
      t.text :body, null: false
      t.text :body_html
      t.timestamps

      t.index :space_id
      t.index :edit_suggestion_id
      t.index :created_user_id
    end

    add_foreign_key :edit_suggestion_comments, :spaces, column: :space_id, validate: false
    add_foreign_key :edit_suggestion_comments, :edit_suggestions, column: :edit_suggestion_id, validate: false
    add_foreign_key :edit_suggestion_comments, :users, column: :created_user_id, validate: false
  end
end
