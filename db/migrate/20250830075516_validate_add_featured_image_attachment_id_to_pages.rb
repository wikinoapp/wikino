class ValidateAddFeaturedImageAttachmentIdToPages < ActiveRecord::Migration[8.0]
  def change
    validate_foreign_key :pages, :attachments, column: :featured_image_attachment_id
  end
end
