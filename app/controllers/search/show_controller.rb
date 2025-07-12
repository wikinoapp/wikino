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
      search_form = Pages::SearchForm.new(
        q: params[:q].to_s.strip
      )

      # 検索結果の取得
      search_results = if search_form.valid? && search_form.q.present?
                         search_pages(search_form.q.not_nil!).to_a
                       else
                         []
                       end

      render_component Search::ShowView.new(
        search_form:,
        search_results:,
        current_user: current_user!
      )
    end

    sig { params(keyword: String).returns(PageRecord::PrivateRelation) }
    private def search_pages(keyword)
      # ユーザーが参加しているスペースのページを検索
      PageRecord
        .joins(space: :space_members)
        .where(space_members: { user_record: current_user_record! })
        .where("page_records.title ILIKE ?", "%#{keyword}%")
        .order(:title)
        .limit(50)
    end
  end
end