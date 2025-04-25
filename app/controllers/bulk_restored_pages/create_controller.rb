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
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      policy = BulkRestoredPagePolicy.new(space_record:, space_member_record:)

      unless policy.create?
        return render_404
      end

      form = TrashedPagesForm.new(form_params.merge(user_record: current_user_record!))

      if form.invalid?
        return render(
          Trash::ShowView.new(
            current_user: UserRepository.new.to_model(user_record: current_user_record!),
            space: SpaceRepository.new.to_model(space_record:),
            page_list: space.restorable_page_list_entity(
              space_viewer:,
              before: params[:before],
              after: params[:after]
            ),
            form:
          ),
          status: :unprocessable_entity
        )
      end

      BulkRestorePagesService.new.call(page_ids: form.page_ids.not_nil!)

      flash[:notice] = t("messages.trash.restored")
      redirect_to trash_path(space.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:trashed_pages_form), ActionController::Parameters).permit(page_ids: [])
    end
  end
end
