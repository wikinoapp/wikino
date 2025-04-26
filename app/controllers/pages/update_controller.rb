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
      page_policy = PagePolicy.new(
        record: page_record,
        user_record: current_user_record!,
        space_member_record:
      )

      unless page_policy.can_update?
        return render_404
      end

      form = EditPageForm.new(form_params.merge(page_record:, space_member_record:))

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

        return render(
          Pages::EditView.new(current_user:, space:, page:, form:, link_list:, backlink_list:), {
            status: :unprocessable_entity
          }
        )
      end

      UpdatePageService.new.call(
        space_member_record:,
        page_record:,
        topic_record: form.topic.not_nil!,
        title: form.title.not_nil!,
        body: form.body.not_nil!
      )

      flash[:notice] = t("messages.page.saved")
      redirect_to page_path(space_record.identifier, page_record.number)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:edit_page_form), ActionController::Parameters).permit(
        :topic_number,
        :title,
        :body
      )
    end
  end
end
