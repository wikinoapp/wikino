# typed: strict
# frozen_string_literal: true

module Spaces
  class DestroyService < ApplicationService
    sig { params(space_record_id: T::Wikino::DatabaseId).void }
    def call(space_record_id:)
      space_record = SpaceRecord.find(space_record_id)

      space_record.topic_records.find_each do |topic_record|
        Topics::DestroyService.new.call(topic_record_id: topic_record.id)
      end

      space_record.export_records.find_each do |export_record|
        Exports::DestroyService.new.call(export_record_id: export_record.id)
      end

      space_record.space_member_records.destroy_all

      space_record.destroy!

      nil
    end
  end
end
