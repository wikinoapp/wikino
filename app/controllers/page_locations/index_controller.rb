# typed: true
# frozen_string_literal: true

module PageLocations
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceAware

    layout false

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = current_space_record
      space_policy = space_policy_for(space_record:)

      unless space_policy.joined_space?
        return render_404
      end

      page_records = space_policy
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
