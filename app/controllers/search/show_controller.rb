# typed: strict
# frozen_string_literal: true

module Search
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      # 検索フォームの初期化
      form = Pages::SearchForm.new(
        q: params[:q].to_s.strip
      )

      # 検索結果の取得
      pages = if form.valid? && form.q.present?
        page_records = search_pages(form.q.not_nil!)
        PageRepository.new.to_models(page_records:)
      else
        []
      end

      render_component Search::ShowView.new(
        form:,
        pages:,
        current_user: current_user!
      )
    end

    sig { params(keyword: String).returns(PageRecord::PrivateRelation) }
    private def search_pages(keyword)
      # ユーザーが参加しているスペースのページを検索
      PageRecord
        .joins(space_record: :space_member_records)
        .where(space_member_records: {user_id: current_user_record!.id})
        .where("pages.title ILIKE ?", "%#{keyword}%")
        .active
        .order(modified_at: :desc)
        .limit(50)
    end
  end
end
