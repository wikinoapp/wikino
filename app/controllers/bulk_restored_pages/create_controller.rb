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
      space_member_policy = SpaceMemberPolicy.new(
        user_record: current_user_record!,
        space_member_record:
      )

      unless space_member_policy.can_create_bulk_restore_pages?
        return render_404
      end

      form = PageForm::BulkRestoring.new(form_params.merge(user_record: current_user_record!))

      if form.invalid?
        current_user = UserRepository.new.to_model(user_record: current_user_record!)
        space = SpaceRepository.new.to_model(space_record:)
        page_list = PageListRepository.new.restorable(
          space_record:,
          before: params[:before],
          after: params[:after]
        )

        return render(Trash::ShowView.new(current_user:, space:, page_list:, form:), {
          status: :unprocessable_entity
        })
      end

      PageService::BulkRestore.new.call(page_ids: form.page_ids.not_nil!)

      flash[:notice] = t("messages.trash.restored")
      redirect_to trash_path(space_record.identifier)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:page_form_bulk_restoring), ActionController::Parameters).permit(page_ids: [])
    end
  end
end
