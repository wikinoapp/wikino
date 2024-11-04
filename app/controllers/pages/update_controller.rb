# typed: true
# frozen_string_literal: true

module Pages
  class UpdateController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::PageSettable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication
    before_action :set_page

    sig { returns(T.untyped) }
    def call
      authorize(@page, :update?)

      @form = EditPageForm.new(form_params.merge(page: @page.not_nil!))

      if @form.invalid?
        @link_collection = @page.not_nil!.fetch_link_collection
        @backlink_collection = @page.not_nil!.fetch_backlink_collection

        return render("pages/edit/call", status: :unprocessable_entity)
      end

      result = UpdatePageUseCase.new.call(
        page: @page.not_nil!,
        topic: @form.topic.not_nil!,
        title: @form.title.not_nil!,
        body: @form.body.not_nil!
      )

      flash[:notice] = t("messages.page.saved")
      redirect_to page_path(Current.space!.identifier, result.page.number)
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
