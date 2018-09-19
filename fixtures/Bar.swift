/**
 * Class: Bar
 * Bar type.
 */
public class Bar {
    /**
     * Static Property: STATIC_PROP
     *
     * Static property.
     */
    public static let STATIC_PROP = "Static property"

    /**
     * Static Method: staticMethod
     * Static method.
     *
     * - Parameter value: The value.
     * - Returns: A Bar instance.
     */
    public static func staticMethod(_ value: String) -> Bar {
        return Bar()
    }

    /**
     * Method: instanceMethod
     * Instance method.
     *
     * - Parameter value: The value.
     * - Returns: The string.
     */
    public func instanceMethod(_ value: String) -> String {
        return value
    }
}
