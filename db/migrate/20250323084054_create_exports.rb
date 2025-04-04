# typed: false
# frozen_string_literal: true

class CreateExports < ActiveRecord::Migration[7.1]
  def change
    create_table :exports, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :queued_by, foreign_key: {to_table: :space_members}, null: false, type: :uuid
      t.timestamps
    end

    create_table :export_statuses, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :export, foreign_key: true, null: false, type: :uuid
      t.integer :kind, null: false
      t.datetime :changed_at, null: false
      t.timestamps
    end

    create_table :export_logs, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :export, foreign_key: true, null: false, type: :uuid
      t.string :message, null: false
      t.datetime :logged_at, null: false
      t.timestamps
    end
  end
end
