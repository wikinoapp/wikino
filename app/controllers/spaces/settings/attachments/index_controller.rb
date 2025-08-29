# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Attachments
      class IndexController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_member_record = current_user_record!.space_member_record(space_record:)
          space_member_policy = SpaceMemberPolicyFactory.build(
            user_record: current_user_record!,
            space_member_record:
          )

          unless space_member_policy.can_update_space?(space_record:)
            return render_404
          end

          # ページネーション
          page = (params[:page] || 1).to_i
          per_page = 50

          search_query = params[:q]
          file_type = params[:file_type]

          attachment_records = space_record.attachment_records
            .preload(active_storage_attachment_record: :blob)
            .order(attached_at: :desc, id: :desc)

          if search_query.present?
            attachment_records = attachment_records
              .joins(active_storage_attachment_record: :blob)
              .where("active_storage_blobs.filename ILIKE ?", "%#{search_query}%")
          end

          if file_type.present?
            attachment_records = attachment_records
              .joins(active_storage_attachment_record: :blob)
              .where("active_storage_blobs.content_type LIKE ?", "#{file_type}/%")
          end

          total_count = attachment_records.count
          attachment_records = attachment_records
            .limit(per_page)
            .offset((page - 1) * per_page)

          space = SpaceRepository.new.to_model(space_record:)
          attachments = attachment_records.map do |attachment_record|
            url = attachment_record.generate_signed_url(space_member_record:)
            AttachmentRepository.new.to_model(
              attachment_record:,
              url:
            )
          end

          render_component Spaces::Settings::Attachments::IndexView.new(
            current_user: current_user!,
            space:,
            attachments:,
            total_count:,
            current_page: page,
            per_page:,
            search_query:
          )
        end
      end
    end
  end
end
