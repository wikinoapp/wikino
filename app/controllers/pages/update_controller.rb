# typed: true
# frozen_string_literal: true

module Pages
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      page_record = space_record.find_page_by_number!(params[:page_number]&.to_i).not_nil!
      space_member_policy = SpaceMemberPolicy.new(
        user_record: current_user_record!,
        space_member_record:
      )

      unless space_member_policy.can_update_page?(page_record:)
        return render_404
      end

      form = Pages::EditForm.new(form_params.merge(page_record:, space_member_record:))

      if form.invalid?
        space = SpaceRepository.new.to_model(space_record:)
        page = PageRepository.new.to_model(page_record:)
        link_list = LinkListRepository.new.to_model(
          user_record: current_user_record,
          pageable_record: page_record
        )
        backlink_list = BacklinkListRepository.new.to_model(
          user_record: current_user_record,
          page_record:
        )

        return render_component(
          Pages::EditView.new(
            current_user: current_user!,
            space:,
            page:,
            form:,
            link_list:,
            backlink_list:
          ),
          status: :unprocessable_entity
        )
      end

      Pages::UpdateService.new.call(
        space_member_record: space_member_record.not_nil!,
        page_record:,
        topic_record: form.topic.not_nil!,
        title: form.title.not_nil!,
        body: form.body.not_nil!
      )

      flash[:notice] = t("messages.pages.saved")
      redirect_to page_path(space_record.identifier, page_record.number)
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
