# typed: strict
# frozen_string_literal: true

class UpdateDraftPageService < ApplicationService
  class Result < T::Struct
    const :draft_page, DraftPageRecord
  end

  sig do
    params(
      space_member: SpaceMemberRecord,
      page: PageRecord,
      topic_number: T.nilable(String),
      title: T.nilable(String),
      body: T.nilable(String)
    ).returns(Result)
  end
  def call(space_member:, page:, topic_number:, title:, body:)
    updated_draft_page = ActiveRecord::Base.transaction do
      draft_page = space_member.find_or_create_draft_page!(page:)
      topic = space_member.topics.find_by(number: topic_number).presence || page.topic
      new_body = body.presence || ""

      draft_page.attributes = {
        topic:,
        title:,
        body: new_body,
        body_html: Markup.new(current_topic: topic.not_nil!).render_html(text: new_body),
        modified_at: Time.zone.now
      }
      draft_page.save!

      draft_page.link!(editor: space_member)

      draft_page
    end

    Result.new(draft_page: updated_draft_page)
  end
end
