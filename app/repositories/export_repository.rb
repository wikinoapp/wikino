# typed: strict
# frozen_string_literal: true

class ExportRepository < ApplicationRepository
  sig { params(export_record: ExportRecord).returns(Export) }
  def to_model(export_record:)
    queued_by = SpaceMemberRepository.new.to_model(space_member_record: export_record.queued_by_record.not_nil!)
    space = SpaceRepository.new.to_model(space_record: export_record.space_record.not_nil!)

    Export.new(database_id: export_record.id, queued_by:, space:)
  end
end
