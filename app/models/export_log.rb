# typed: strict
# frozen_string_literal: true

class ExportLog < ApplicationRecord
  belongs_to :space
  belongs_to :export

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(ExportLogEntity) }
  def to_entity(space_viewer:)
    ExportLogEntity.new(
      database_id: id,
      message:,
      logged_at:,
      space_entity: space.not_nil!.to_entity(space_viewer:),
      export_entity: export.to_entity(space_viewer:)
    )
  end
end
