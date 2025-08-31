# typed: true
# frozen_string_literal: true

module DraftPages
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      page_record = space_record.find_page_by_number!(params[:page_number]&.to_i)
      topic_policy = topic_policy_for(topic_record: page_record.topic_record.not_nil!)

      unless topic_policy.can_update_draft_page?(page_record:)
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)
      result = DraftPages::UpdateService.new.call(
        space_member_record: space_member_record.not_nil!,
        page_record:,
        topic_number: form_params[:topic_number],
        title: form_params[:title],
        body: form_params[:body]
      )

      draft_page_record = result.draft_page_record
      draft_page = DraftPageRepository.new.to_model(draft_page_record:)
      link_list = LinkListRepository.new.to_model(
        user_record: current_user_record!,
        pageable_record: draft_page_record
      )
      backlink_list = BacklinkListRepository.new.to_model(
        user_record: current_user_record!,
        page_record:
      )

      render_component(
        DraftPages::UpdateView.new(
          current_user: current_user!,
          draft_page:,
          link_list:,
          backlink_list:
        ),
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      )
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:pages_edit_form), ActionController::Parameters).permit(
        :topic_number,
        :title,
        :body
      )
    end
  end
end
