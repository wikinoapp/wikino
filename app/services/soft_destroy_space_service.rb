# typed: strict
# frozen_string_literal: true

class SoftDestroySpaceService < ApplicationService
  sig { params(space_record: SpaceRecord).void }
  def call(space_record:)
    ActiveRecord::Base.transaction do
      space_record.topic_records.discard_all
      space_record.discard!
    end

    DestroySpaceJob.perform_later(space_record_id: space_record.id)

    nil
  end
end
