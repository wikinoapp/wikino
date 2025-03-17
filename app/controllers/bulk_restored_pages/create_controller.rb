# typed: true
# frozen_string_literal: true

module BulkRestoredPages
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = Space.find_by_identifier!(params[:space_identifier])
      space_viewer = Current.viewer!.space_viewer!(space:)

      unless space_viewer.can_create_bulk_restored_pages?(space:)
        return render_404
      end

      form = TrashedPagesForm.new(form_params)
      form.user = T.let(Current.viewer!, User)

      if form.invalid?
        return render(
          Trash::ShowView.new(
            current_user_entity: Current.viewer!.user_entity,
            space_entity: space.to_entity(space_viewer:),
            page_list_entity: space.restorable_page_list_entity(
              space_viewer:,
              before: params[:before],
              after: params[:after]
            ),
            form:
          ),
          status: :unprocessable_entity
        )
      end

      BulkRestorePagesUseCase.new.call(page_ids: form.page_ids.not_nil!)

      flash[:notice] = t("messages.trash.restored")
      redirect_to trash_path(space.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:trashed_pages_form), ActionController::Parameters).permit(page_ids: [])
    end
  end
end
