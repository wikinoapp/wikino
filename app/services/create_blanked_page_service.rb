# typed: strict
# frozen_string_literal: true

class CreateBlankedPageService < ApplicationService
  class Result < T::Struct
    const :page, Page
  end

  sig { params(topic: TopicRecord).returns(Result) }
  def call(topic:)
    space_member = Current.viewer!.active_space_members.find_by!(space_id: topic.space_id)

    page = ActiveRecord::Base.transaction do
      new_page = PageRecord.create_as_blanked!(topic:)
      new_page.add_editor!(editor: space_member)
      new_page
    end

    Result.new(page:)
  end
end
