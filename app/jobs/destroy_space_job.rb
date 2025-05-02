# typed: strict
# frozen_string_literal: true

class DestroySpaceJob < ApplicationJob
  queue_as :low

  sig { params(space_record_id: T::Wikino::DatabaseId).void }
  def perform(space_record_id:)
    SpaceService::Destroy.new.call(space_record_id:)
  end
end
