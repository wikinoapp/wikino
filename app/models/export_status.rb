# typed: strict
# frozen_string_literal: true

class ExportStatus < ApplicationRecord
  enum :kind, {
    ExportStatusKind::Queued.serialize => 0,
    ExportStatusKind::Started.serialize => 1,
    ExportStatusKind::Succeeded.serialize => 2,
    ExportStatusKind::Failed.serialize => 3
  }, prefix: true

  belongs_to :space
  belongs_to :export

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(ExportStatusEntity) }
  def to_entity(space_viewer:)
    ExportStatusEntity.new(
      database_id: id,
      kind:,
      changed_at:,
      space_entity: space.not_nil!.to_entity(space_viewer:),
      export_entity: export.to_entity(space_viewer:)
    )
  end
end
