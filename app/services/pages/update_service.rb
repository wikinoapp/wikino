# typed: strict
# frozen_string_literal: true

module Pages
  class UpdateService < ApplicationService
    class Result < T::Struct
      const :page_record, PageRecord
    end

    sig do
      params(
        space_member_record: SpaceMemberRecord,
        page_record: PageRecord,
        topic_record: TopicRecord,
        title: String,
        body: String
      ).returns(Result)
    end
    def call(space_member_record:, page_record:, topic_record:, title:, body:)
      now = Time.current

      page_record.attributes = {
        topic_record:,
        title:,
        body:,
        body_html: Markup.new(current_topic: topic_record, current_space_member: space_member_record).render_html(text: body),
        modified_at: now
      }
      page_record.published_at = now if page_record.published_at.nil?

      updated_page_record = ActiveRecord::Base.transaction do
        page_record.save!
        page_record.add_editor!(editor_record: space_member_record)
        page_record.create_revision!(editor_record: space_member_record, body:, body_html: body)
        page_record.link!(editor_record: space_member_record)
        space_member_record.destroy_draft_page!(page_record:)

        page_record
      end

      Result.new(page_record: updated_page_record)
    end
  end
end
