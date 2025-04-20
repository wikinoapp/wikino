# typed: true
# frozen_string_literal: true

# ログインしていない人を表すモデル
class Visitor < ApplicationModel
  include ModelConcerns::Viewable

  sig { params(attributes: T.untyped).void }
  def initialize(attributes = {})
    super
    @serialized_locale ||= "ja"
    @time_zone ||= "Asia/Tokyo"
  end

  sig { override.returns(String) }
  attr_accessor :serialized_locale

  sig { override.returns(String) }
  attr_accessor :time_zone

  sig { override.returns(T::Boolean) }
  def signed_in?
    false
  end
end
