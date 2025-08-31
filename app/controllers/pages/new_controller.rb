# typed: true
# frozen_string_literal: true

module Pages
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      topic_record = space_record.find_topic_by_number!(params[:topic_number]&.to_i)
      topic_policy = topic_policy_for(topic_record:)

      unless topic_policy.can_create_page?(topic_record:)
        return render_404
      end

      result = Pages::CreateBlankedService.new.call(
        topic_record:,
        editor_record: space_member_record.not_nil!
      )

      redirect_to edit_page_path(space_record.identifier, result.page_record.number)
    end
  end
end
