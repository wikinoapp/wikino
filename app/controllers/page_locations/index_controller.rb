# typed: true
# frozen_string_literal: true

module PageLocations
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    layout false

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record&.space_member_record(space_record:)
      space_member_policy = SpacePolicyFactory.build(
        user_record: current_user_record,
        space_member_record:
      )

      unless space_member_policy.joined_space?
        return render_404
      end

      page_records = space_member_policy
        .showable_pages(space_record:)
        .filter_by_title(q: params[:q])
        .order(modified_at: :desc)
        .limit(10)

      render json: {
        page_locations: page_records.map { |page_record|
          {
            key: "#{page_record.topic_record.not_nil!.name}/#{page_record.title}"
          }
        }
      }
    end
  end
end
