# typed: strict
# frozen_string_literal: true

module DraftPages
  class UpdateService < ApplicationService
    class Result < T::Struct
      const :draft_page_record, DraftPageRecord
    end

    sig do
      params(
        space_member_record: SpaceMemberRecord,
        page_record: PageRecord,
        topic_number: T.nilable(String),
        title: T.nilable(String),
        body: T.nilable(String)
      ).returns(Result)
    end
    def call(space_member_record:, page_record:, topic_number:, title:, body:)
      updated_draft_page = ActiveRecord::Base.transaction do
        draft_page = space_member_record.find_or_create_draft_page!(page: page_record)
        topic = space_member_record.topic_records.find_by(number: topic_number).presence || page_record.topic_record
        new_body = body.presence || ""

        draft_page.attributes = {
          topic_record: topic,
          title:,
          body: new_body,
          body_html: Markup.new(current_topic: topic.not_nil!).render_html(text: new_body),
          modified_at: Time.zone.now
        }
        draft_page.save!

        draft_page.link!(editor_record: space_member_record)

        draft_page
      end

      Result.new(draft_page_record: updated_draft_page)
    end
  end
end
