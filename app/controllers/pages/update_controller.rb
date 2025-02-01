# typed: true
# frozen_string_literal: true

module Pages
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      page = space.find_page_by_number!(params[:page_number]&.to_i).not_nil!

      unless Current.viewer!.can_update_page?(page:)
        return render_404
      end

      form = EditPageForm.new(form_params.merge(page:))

      if form.invalid?
        link_collection = page.fetch_link_collection
        backlink_collection = page.fetch_backlink_collection

        return render(
          Pages::EditView.new(space:, page:, form:, link_collection:, backlink_collection:),
          status: :unprocessable_entity
        )
      end

      result = UpdatePageUseCase.new.call(
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
