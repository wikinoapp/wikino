# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Localizable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { params(action: Proc).returns(T.untyped) }
    def set_locale(&action)
      I18n.with_locale(current_locale.serialize, &action)
    end

    sig(:final) { returns(T.nilable(ViewerLocale)) }
    private def instant_locale
      @instant_locale ||= T.let(ViewerLocale.try_deserialize(params[:locale]), T.nilable(ViewerLocale))
    end

    sig(:final) { returns(ViewerLocale) }
    private def preferred_locale
      preferred_languages = http_accept_language.user_preferred_languages
      # Chrome returns "ja", but Safari would return "ja-JP", not "ja".
      (preferred_languages.present? && preferred_languages.all? { |lang| !lang.match?(/ja/) }) ? ViewerLocale::En : default_locale
    end

    sig(:final) { returns(ViewerLocale) }
    private def default_locale
      ViewerLocale::Ja
    end

    sig(:final) { returns(ViewerLocale) }
    private def current_locale
      instant_locale.presence || Current.viewer&.locale.presence || preferred_locale
    end
  end
end
