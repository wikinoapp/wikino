# typed: strict
# frozen_string_literal: true

class CreateBlankedPageService < ApplicationService
  class Result < T::Struct
    const :page_record, PageRecord
  end

  sig { params(topic_record: TopicRecord, editor_record: SpaceMemberRecord).returns(Result) }
  def call(topic_record:, editor_record:)
    page_record = ActiveRecord::Base.transaction do
      new_page_record = PageRecord.create_as_blanked!(topic_record:)
      new_page_record.add_editor!(editor_record:)
      new_page_record
    end

    Result.new(page_record:)
  end
end
