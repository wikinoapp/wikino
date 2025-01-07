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
      form.user = Current.user!

      if form.invalid?
        return render_error_view(form:)
      end

      BulkRestorePagesUseCase.new.call(page_ids: form.page_ids.not_nil!)

      flash[:notice] = t("messages.trash.restored")
      redirect_to trash_path(Current.space!.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:trashed_pages_form), ActionController::Parameters).permit(page_ids: [])
    end

    sig { params(form: TrashedPagesForm).returns(ActiveSupport::SafeBuffer) }
    private def render_error_view(form:)
      error_view = Trash::ShowView.new(
        page_connection: Page.restorable_connection(before: params[:before], after: params[:after]),
        form:
      )
      render(error_view, status: :unprocessable_entity)
    end
  end
end
