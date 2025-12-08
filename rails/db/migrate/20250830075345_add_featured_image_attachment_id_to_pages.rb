class AddFeaturedImageAttachmentIdToPages < ActiveRecord::Migration[8.0]
  disable_ddl_transaction!

  def change
    add_column :pages, :featured_image_attachment_id, :uuid, null: true
    add_foreign_key :pages, :attachments, column: :featured_image_attachment_id, validate: false
    add_index :pages, :featured_image_attachment_id, algorithm: :concurrently
  end
end
