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
      space = Space.find_by_identifier!(params[:space_identifier])
      space_viewer = Current.viewer!.space_viewer!(space:)
      page = space.find_page_by_number!(params[:page_number]&.to_i).not_nil!

      unless space_viewer.can_update_page?(page:)
        return render_404
      end

      form = EditPageForm.new(
        form_params.merge(
          page:,
          space_member: T.let(space_viewer, SpaceMember)
        )
      )

      if form.invalid?
        link_list_entity = page.fetch_link_list_entity(space_viewer:)
        backlink_list_entity = page.fetch_backlink_list_entity(space_viewer:)

        return render(
          Pages::EditView.new(
            current_user_entity: Current.viewer!.user_entity,
            space_entity: space.to_entity(space_viewer:),
            page_entity: page.to_entity(space_viewer:),
            form:,
            link_list_entity:,
            backlink_list_entity:
          ),
          status: :unprocessable_entity
        )
      end

      result = UpdatePageService.new.call(
        page:,
        topic: form.topic.not_nil!,
        title: form.title.not_nil!,
        body: form.body.not_nil!
      )

      flash[:notice] = t("messages.page.saved")
      redirect_to page_path(space.identifier, result.page.number)
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
