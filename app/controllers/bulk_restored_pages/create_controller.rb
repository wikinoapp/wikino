# typed: true
# frozen_string_literal: true

module BulkRestoredPages
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!

      form = TrashedPagesForm.new(form_params)
      form.user = Current.user!

      if form.invalid?
        return render_error_view(space:, form:)
      end

      BulkRestorePagesUseCase.new.call(page_ids: form.page_ids.not_nil!)

      flash[:notice] = t("messages.trash.restored")
      redirect_to trash_path(space.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:trashed_pages_form), ActionController::Parameters).permit(page_ids: [])
    end

    sig { params(space: Space, form: TrashedPagesForm).returns(ActiveSupport::SafeBuffer) }
    private def render_error_view(space:, form:)
      error_view = Trash::ShowView.new(
        page_connection: space.restorable_page_connection(before: params[:before], after: params[:after]),
        form:
      )
      render(error_view, status: :unprocessable_entity)
    end
  end
end
