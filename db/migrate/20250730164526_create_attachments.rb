# typed: false
# frozen_string_literal: true

class CreateAttachments < ActiveRecord::Migration[8.0]
  def change
    create_table :attachments, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :space, type: :uuid, null: false, foreign_key: true
      t.references :active_storage_attachment, type: :uuid, null: false, foreign_key: true
      t.references :attached_user, type: :uuid, null: false, foreign_key: {to_table: :users}
      t.datetime :attached_at, null: false
      t.timestamps
    end

    add_index :attachments, :attached_at
  end
end
