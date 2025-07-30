# typed: false
# frozen_string_literal: true

class CreatePageAttachmentReferences < ActiveRecord::Migration[8.0]
  def change
    create_table :page_attachment_references, id: false do |t|
      t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
      t.references :attachment, type: :uuid, null: false, foreign_key: true
      t.references :page, type: :uuid, null: false, foreign_key: true
      t.timestamps
    end

    add_index :page_attachment_references, %i[page_id attachment_id], unique: true
  end
end
