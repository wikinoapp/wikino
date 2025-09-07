# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe Attachments::ProcessService do
  describe "#call" do
    it "処理が必要ない場合はスキップすること" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :completed)

      service = Attachments::ProcessService.new
      result = service.call(attachment_record:)

      expect(result.success).to eq(true)
      expect(attachment_record.processing_status_completed?).to eq(true)
    end

    it "blobがない場合は失敗すること" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :pending)
      allow(attachment_record).to receive(:blob_record).and_return(nil)

      service = Attachments::ProcessService.new
      result = service.call(attachment_record:)

      expect(result.success).to eq(false)
      expect(attachment_record.processing_status_failed?).to eq(true)
    end

    it "S3にファイルが存在しない場合はpendingに戻すこと" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :pending)
      blob_record = attachment_record.blob_record

      # S3にファイルが存在しないことをシミュレート
      allow(blob_record.service).to receive(:exist?).with(blob_record.key).and_return(false)

      service = Attachments::ProcessService.new
      result = service.call(attachment_record:)

      expect(result.success).to eq(false)
      expect(attachment_record.processing_status_pending?).to eq(true)
    end

    it "画像処理に失敗した場合はpendingに戻すこと" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :pending)
      blob_record = attachment_record.blob_record

      # S3にファイルは存在するが、画像処理に失敗
      allow(blob_record.service).to receive(:exist?).with(blob_record.key).and_return(true)
      allow(blob_record).to receive(:image?).and_return(true)
      allow(blob_record).to receive(:process_image_with_exif_removal).and_return(false)

      service = Attachments::ProcessService.new
      result = service.call(attachment_record:)

      expect(result.success).to eq(false)
      expect(attachment_record.processing_status_pending?).to eq(true)
    end

    it "正常に処理が完了した場合はcompletedにすること" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :pending)
      blob_record = attachment_record.blob_record

      # 正常な処理をシミュレート
      allow(blob_record.service).to receive(:exist?).with(blob_record.key).and_return(true)
      allow(blob_record).to receive(:image?).and_return(true)
      allow(blob_record).to receive(:process_image_with_exif_removal).and_return(true)

      service = Attachments::ProcessService.new
      result = service.call(attachment_record:)

      expect(result.success).to eq(true)
      expect(attachment_record.processing_status_completed?).to eq(true)
    end

    it "例外が発生した場合はfailedにすること" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :pending)
      blob_record = attachment_record.blob_record

      # 例外を発生させる
      allow(blob_record.service).to receive(:exist?).and_raise(StandardError, "Test error")

      service = Attachments::ProcessService.new

      expect {
        service.call(attachment_record:)
      }.to raise_error(StandardError, "Test error")

      expect(attachment_record.processing_status_failed?).to eq(true)
    end
  end
end

