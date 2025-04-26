# typed: strict
# frozen_string_literal: true

class ExportStatusRepository < ApplicationRepository
  sig { params(export_status_record: ExportStatusRecord).returns(ExportStatus) }
  def to_model(export_status_record:)
    ExportStatus.new(
      database_id: export_status_record.id,
      kind: ExportStatusKind.deserialize(export_status_record.kind),
      changed_at: export_status_record.changed_at,
      space: SpaceRepository.new.to_model(space_record: export_status_record.space_record.not_nil!),
      export: ExportRepository.new.to_model(export_record: export_status_record.export_record.not_nil!)
    )
  end
end
