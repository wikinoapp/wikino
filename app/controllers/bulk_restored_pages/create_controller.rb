# typed: true
# frozen_string_literal: true

module BulkRestoredPages
  class CreateController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      form = TrashedPagesForm.new(form_params)

      if form.invalid?
        error_view = Views::Trash::Show.new(
          page_connection: Page.restorable_connection(before: params[:before], after: params[:after]),
          form:
        )
        return render(error_view, status: :unprocessable_entity)
      end

      flash[:notice] = t("messages.trash.restored")
      redirect_to trash_path(Current.space!.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:trashed_pages_form), ActionController::Parameters).permit(page_ids: [])
    end
  end
end
