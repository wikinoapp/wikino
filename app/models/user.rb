# typed: false
# frozen_string_literal: true

class User < ApplicationRecord
  extend T::Sig

  include SoftDeletable

  has_many :notes, dependent: :destroy

  # ログイン情報を記録する
  # Deviseの #update_tracked_fields を参考にしている
  # https://github.com/heartcombo/devise/blob/451ff6d49c71e543962d2b29d77f2e744b2d47e1/lib/devise/models/trackable.rb#L20-L31
  def track_sign_in!(request)
    old_current, new_current = current_signed_in_at, Time.now.utc
    self.last_signed_in_at = old_current || new_current
    self.current_signed_in_at = new_current

    self.sign_in_count += 1

    save!(validate: false)
  end
end
