# typed: strict
# frozen_string_literal: true

class UpdatePageUseCase < ApplicationUseCase
  class Result < T::Struct
    const :page, Page
  end

  sig { params(page: Page, topic: Topic, title: String, body: String).returns(Result) }
  def call(page:, topic:, title:, body:)
    now = Time.zone.now

    page.attributes = {
      topic:,
      title:,
      body:,
      body_html: Markup.new(current_topic: topic).render_html(text: body),
      modified_at: now
    }
    page.published_at = now if page.published_at.nil?
    space_member = Current.viewer!.active_space_members.find_by!(space_id: page.space_id)

    updated_page = ActiveRecord::Base.transaction do
      page.save!
      page.add_editor!(editor: space_member)
      page.create_revision!(editor: space_member, body:, body_html: body)
      page.link!(editor: space_member)
      space_member.destroy_draft_page!(page:)

      page
    end

    Result.new(page: updated_page)
  end
end
