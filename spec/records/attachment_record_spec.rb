# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe AttachmentRecord do
  describe "#thumbnail_variant" do
    it "カードサイズのバリアントを生成できること" do
      # 画像ファイルを持つ添付ファイルレコードを作成
      attachment = FactoryBot.create(:attachment_record, :with_image)

      # thumbnail_variantを呼び出し
      variant = attachment.thumbnail_variant(size: AttachmentThumbnailSize::Card)

      # variantが返されることを確認
      expect(variant).to be_present
      expect(variant).to be_a(ActiveStorage::Variant).or be_a(ActiveStorage::VariantWithRecord)
    end

    it "OGサイズのバリアントを生成できること" do
      attachment = FactoryBot.create(:attachment_record, :with_image)

      variant = attachment.thumbnail_variant(size: AttachmentThumbnailSize::Og)

      expect(variant).to be_present
      expect(variant).to be_a(ActiveStorage::Variant).or be_a(ActiveStorage::VariantWithRecord)
    end

    it "画像でないファイルの場合はnilを返すこと" do
      # PDFファイルを持つ添付ファイルレコードを作成
      attachment = FactoryBot.create(:attachment_record, :with_pdf)

      variant = attachment.thumbnail_variant(size: AttachmentThumbnailSize::Card)

      expect(variant).to be_nil
    end

    it "blobがない場合はnilを返すこと" do
      attachment = FactoryBot.build(:attachment_record)
      allow(attachment).to receive(:blob_record).and_return(nil)

      variant = attachment.thumbnail_variant(size: AttachmentThumbnailSize::Card)

      expect(variant).to be_nil
    end
  end

  describe "#thumbnail_url" do
    it "カードサイズのサムネイルURLを生成できること" do
      attachment = FactoryBot.create(:attachment_record, :with_image)

      # variantがprocessedを返すようにモック
      variant = instance_double(ActiveStorage::Variant)
      processed_variant = instance_double(ActiveStorage::VariantWithRecord)
      allow(attachment).to receive(:thumbnail_variant).with(size: AttachmentThumbnailSize::Card).and_return(variant)
      allow(variant).to receive(:processed).and_return(processed_variant)
      allow(processed_variant).to receive(:url).and_return("http://example.com/rails/active_storage/variant/123")

      url = attachment.thumbnail_url(size: AttachmentThumbnailSize::Card)

      expect(url).to be_present
      expect(url).to be_a(String)
      expect(url).to include("rails/active_storage")
    end

    it "OGサイズのサムネイルURLを生成できること" do
      attachment = FactoryBot.create(:attachment_record, :with_image)

      # variantがprocessedを返すようにモック
      variant = instance_double(ActiveStorage::Variant)
      processed_variant = instance_double(ActiveStorage::VariantWithRecord)
      allow(attachment).to receive(:thumbnail_variant).with(size: AttachmentThumbnailSize::Og).and_return(variant)
      allow(variant).to receive(:processed).and_return(processed_variant)
      allow(processed_variant).to receive(:url).and_return("http://example.com/rails/active_storage/variant/456")

      url = attachment.thumbnail_url(size: AttachmentThumbnailSize::Og)

      expect(url).to be_present
      expect(url).to be_a(String)
      expect(url).to include("rails/active_storage")
    end

    it "有効期限を指定できること" do
      attachment = FactoryBot.create(:attachment_record, :with_image)

      # variantがprocessedを返すようにモック
      variant = instance_double(ActiveStorage::Variant)
      processed_variant = instance_double(ActiveStorage::VariantWithRecord)
      allow(attachment).to receive(:thumbnail_variant).with(size: AttachmentThumbnailSize::Card).and_return(variant)
      allow(variant).to receive(:processed).and_return(processed_variant)
      allow(processed_variant).to receive(:url).with(expires_in: 2.hours).and_return("http://example.com/rails/active_storage/variant/789?expires_in=7200")

      url = attachment.thumbnail_url(size: AttachmentThumbnailSize::Card, expires_in: 2.hours)

      expect(url).to be_present
      # URLに有効期限のパラメータが含まれることを確認
      expect(url).to include("expires_in")
    end

    it "画像でないファイルの場合はnilを返すこと" do
      attachment = FactoryBot.create(:attachment_record, :with_pdf)

      url = attachment.thumbnail_url(size: AttachmentThumbnailSize::Card)

      expect(url).to be_nil
    end

    it "エラーが発生した場合はnilを返すこと" do
      attachment = FactoryBot.create(:attachment_record, :with_image)
      # variantがエラーを起こすようにモック
      allow(attachment).to receive(:thumbnail_variant).and_raise(StandardError.new("Test error"))

      url = attachment.thumbnail_url(size: AttachmentThumbnailSize::Card)

      expect(url).to be_nil
    end
  end

  describe "#generate_thumbnails" do
    it "廃止されたメソッドなので常にtrueを返すこと" do
      attachment = FactoryBot.create(:attachment_record, :with_image)

      result = attachment.generate_thumbnails

      expect(result).to be true
    end

    it "画像でないファイルでもtrueを返すこと" do
      attachment = FactoryBot.create(:attachment_record, :with_pdf)

      result = attachment.generate_thumbnails

      expect(result).to be true
    end
  end
end
