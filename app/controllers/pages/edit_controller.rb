# typed: true
# frozen_string_literal: true

module Pages
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      page_record = space_record.find_page_by_number!(params[:page_number]&.to_i)
      space_member_policy = SpacePolicyFactory.build(
        user_record: current_user_record!,
        space_member_record:
      )

      unless space_member_policy.can_update_page?(page_record:)
        return render_404
      end

      draft_page_record = space_member_record.not_nil!.draft_page_records.find_by(page_record:)
      pageable_record = draft_page_record.presence || page_record

      space = SpaceRepository.new.to_model(space_record:)
      page = PageRepository.new.to_model(page_record:, current_space_member: space_member_record)

      form = Pages::EditForm.new(
        space_member_record:,
        topic_number: pageable_record.topic_record.not_nil!.number,
        title: pageable_record.title,
        body: pageable_record.body
      )

      link_list = LinkListRepository.new.to_model(user_record: current_user_record, pageable_record:)
      backlink_list = BacklinkListRepository.new.to_model(user_record: current_user_record, page_record:)
      current_user = UserRepository.new.to_model(user_record: current_user_record!)

      render_component Pages::EditView.new(
        space:,
        page:,
        draft_page: draft_page(draft_page_record:),
        form:,
        link_list:,
        backlink_list:,
        current_user:
      )
    end

    sig { params(draft_page_record: T.nilable(DraftPageRecord)).returns(T.nilable(DraftPage)) }
    private def draft_page(draft_page_record:)
      return if draft_page_record.nil?

      DraftPageRepository.new.to_model(draft_page_record: draft_page_record.not_nil!)
    end
  end
end
