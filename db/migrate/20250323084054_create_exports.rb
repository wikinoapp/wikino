# typed: false
# frozen_string_literal: true

class CreateExports < ActiveRecord::Migration[7.1]
  def change
    create_table :exports, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :started_by, foreign_key: {to_table: :space_members}, null: false, type: :uuid
      t.datetime :started_at, null: false
      t.datetime :finished_at
      # t.datetime :failed_at
      t.timestamps
    end

    create_table :export_logs, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, foreign_key: true, null: false, type: :uuid
      t.references :export, foreign_key: true, null: false, type: :uuid
      t.datetime :logged_at, null: false
      t.string :message, null: false
      t.timestamps
    end
  end
end
