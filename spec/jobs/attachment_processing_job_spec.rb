# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe AttachmentProcessingJob, type: :job do
  describe "#perform" do
    it "存在しないattachment_recordの場合は何もしないこと" do
      service = instance_double(Attachments::ProcessService)
      allow(Attachments::ProcessService).to receive(:new).and_return(service)
      allow(service).to receive(:call)

      job = AttachmentProcessingJob.new
      job.perform("non-existent-id")

      expect(service).not_to have_received(:call)
    end

    it "処理が成功した場合はリトライしないこと" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :pending)
      result = Attachments::ProcessService::Result.new(attachment_record:, success: true)

      service = instance_double(Attachments::ProcessService)
      allow(Attachments::ProcessService).to receive(:new).and_return(service)
      allow(service).to receive(:call).with(attachment_record:).and_return(result)
      allow(AttachmentProcessingJob).to receive(:set)

      job = AttachmentProcessingJob.new
      job.perform(attachment_record.id)

      expect(AttachmentProcessingJob).not_to have_received(:set)
    end

    it "処理が失敗しpendingステータスの場合はリトライすること" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :pending)
      result = Attachments::ProcessService::Result.new(attachment_record:, success: false)

      service = instance_double(Attachments::ProcessService)
      allow(Attachments::ProcessService).to receive(:new).and_return(service)
      allow(service).to receive(:call).with(attachment_record:).and_return(result)

      # リトライジョブがスケジュールされることを確認
      job_wrapper = instance_double(ActiveJob::ConfiguredJob)
      allow(AttachmentProcessingJob).to receive(:set).with(wait: 5.seconds).and_return(job_wrapper)
      allow(job_wrapper).to receive(:perform_later)

      job = AttachmentProcessingJob.new
      job.perform(attachment_record.id)

      expect(job_wrapper).to have_received(:perform_later).with(attachment_record.id)
    end

    it "処理が失敗しfailedステータスの場合はリトライしないこと" do
      attachment_record = FactoryBot.create(:attachment_record, processing_status: :failed)
      result = Attachments::ProcessService::Result.new(attachment_record:, success: false)

      service = instance_double(Attachments::ProcessService)
      allow(Attachments::ProcessService).to receive(:new).and_return(service)
      allow(service).to receive(:call).with(attachment_record:).and_return(result)
      allow(AttachmentProcessingJob).to receive(:set)

      job = AttachmentProcessingJob.new
      job.perform(attachment_record.id)

      expect(AttachmentProcessingJob).not_to have_received(:set)
    end
  end
end
