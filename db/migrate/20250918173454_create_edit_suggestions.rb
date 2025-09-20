# typed: false
# frozen_string_literal: true

class CreateEditSuggestions < ActiveRecord::Migration[8.0]
  def change
    create_table :edit_suggestions, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, type: :uuid, null: false, foreign_key: true
      t.references :topic, type: :uuid, null: false, foreign_key: true
      t.references :created_user, type: :uuid, null: false, foreign_key: {to_table: :users}
      t.string :title, null: false
      t.text :description, null: false
      t.integer :status, null: false, default: 0
      t.datetime :applied_at
      t.timestamps

      t.index :status
      t.index [:topic_id, :status]
    end
  end
end
