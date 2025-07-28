# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::GenerateExportFilesService, type: :service do
  describe "#call" do
    it "エクスポートが失敗しているとき、何もしないこと" do
      export_record = create(:export_record, :failed)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Failed)

      Spaces::GenerateExportFilesService.new.call(export_record:)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Failed)
    end

    it "エクスポートが成功しているとき、何もしないこと" do
      export_record = create(:export_record, :succeeded)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Succeeded)

      Spaces::GenerateExportFilesService.new.call(export_record:)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Succeeded)
    end

    it "エクスポートが開始されていないとき、エクスポートを開始すること" do
      export_record = create(:export_record, :queued)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Queued)

      Spaces::GenerateExportFilesService.new.call(export_record:)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.order(changed_at: :asc).pluck(:kind)).to eq([
        ExportStatusKind::Queued.serialize,
        ExportStatusKind::Started.serialize,
        ExportStatusKind::Succeeded.serialize
      ])
    end
  end
end
