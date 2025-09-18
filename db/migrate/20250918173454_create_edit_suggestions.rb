# typed: false
# frozen_string_literal: true

class CreateEditSuggestions < ActiveRecord::Migration[8.0]
  def change
    create_table :edit_suggestions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.uuid :space_id, null: false
      t.uuid :topic_id, null: false
      t.uuid :created_user_id, null: false
      t.string :title, null: false
      t.text :description
      t.string :status, null: false, default: "draft"
      t.datetime :applied_at
      t.timestamps

      t.index :space_id
      t.index :topic_id
      t.index :created_user_id
      t.index :status
      t.index [:topic_id, :status]
    end

    add_foreign_key :edit_suggestions, :spaces, column: :space_id, validate: false
    add_foreign_key :edit_suggestions, :topics, column: :topic_id, validate: false
    add_foreign_key :edit_suggestions, :users, column: :created_user_id, validate: false
  end
end
