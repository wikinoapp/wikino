# typed: strict
# frozen_string_literal: true

class Page < ApplicationModel
  # ページをゴミ箱に移動してから削除されるまでの日数
  DELETE_LIMIT_DAYS = 30
  # タイトルの最大文字数 (値に強い理由は無い)
  TITLE_MAX_LENGTH = 200
end
