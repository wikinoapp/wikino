class AddProcessingStatusToAttachmentRecords < ActiveRecord::Migration[8.0]
  disable_ddl_transaction!

  def change
    add_column :attachments, :processing_status, :integer, null: false, default: 0
    add_index :attachments, :processing_status, algorithm: :concurrently
  end
end
