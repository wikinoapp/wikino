# typed: true
# frozen_string_literal: true

module Pages
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      topic_record = space_record.topic_records.kept.find_by!(number: params[:topic_number])
      topic_policy = TopicPolicy.new(record: topic_record, space_member_record:)

      unless topic_policy.create_page?
        return render_404
      end

      result = CreateBlankedPageService.new.call(
        topic_record:,
        editor_record: space_member_record.not_nil!
      )

      redirect_to edit_page_path(space_record.identifier, result.page_record.number)
    end
  end
end
