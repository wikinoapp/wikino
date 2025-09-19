# typed: false
# frozen_string_literal: true

class CreateEditSuggestionComments < ActiveRecord::Migration[8.0]
  def change
    create_table :edit_suggestion_comments, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, type: :uuid, null: false, foreign_key: true
      t.references :edit_suggestion, type: :uuid, null: false, foreign_key: true
      t.references :created_user, type: :uuid, null: false, foreign_key: {to_table: :users}
      t.text :body, null: false
      t.text :body_html
      t.timestamps
    end
  end
end
